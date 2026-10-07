package npf

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// renderInstr is a deferred render step for one parsed top-level block.
type renderInstr struct {
	block  any
	render func() *Node
}

// Formatter renders parsed NPF blocks into HTML.
type Formatter struct {
	content []any
	layout  []any

	localizer             Localizer
	urlHandler            func(string) string
	forbidExternalIframes bool
	truncate              bool

	idx                   int
	current               any
	currentContextPadding int
	renderInstructions    []*renderInstr
	hasRenderError        bool
	post                  *Node

	// Hooks let hyperblur augment the base output (download buttons, poll
	// data attributes, ...) without reimplementing the formatter.
	ImageHook func(f *Formatter, block *ImageBlock, rowLength int, overrideAspect *float64, node *Node) *Node
	VideoHook func(f *Formatter, block *VideoBlock, node *Node) *Node
	PollHook  func(f *Formatter, block *PollBlock, node *Node) *Node

	// Context hyperblur needs when augmenting output.
	BlogName string
	PostID   string
}

func (f *Formatter) next() bool {
	if f.idx >= len(f.content) {
		f.current = nil
		return false
	}
	f.current = f.content[f.idx]
	f.idx++
	return true
}

// HasRenderError reports whether any block failed to render cleanly.
func (f *Formatter) HasRenderError() bool { return f.hasRenderError }

// URLHandler returns the configured URL handler.
func (f *Formatter) URLHandler(url string) string { return f.urlHandler(url) }

// ---------------------------------------------------------------------------
// Block formatting
// ---------------------------------------------------------------------------

func (f *Formatter) prepare() *renderInstr {
	switch b := f.current.(type) {
	case *TextBlock:
		if len(b.Nest) > 0 {
			f.currentContextPadding = calculatePad(b)
		}
		return &renderInstr{block: b, render: func() *Node { return formatText(b, f.urlHandler) }}
	case *ListGrouping:
		f.currentContextPadding = calculatePad(b)
		return &renderInstr{block: b, render: func() *Node { return f.formatList(b) }}
	case *ImageBlock:
		return &renderInstr{block: b, render: func() *Node { return f.formatImage(b, 1, nil) }}
	case *LinkBlock:
		return &renderInstr{block: b, render: func() *Node { return f.formatLink(b) }}
	case *AudioBlock:
		return &renderInstr{block: b, render: func() *Node { return f.formatAudio(b) }}
	case *VideoBlock:
		return &renderInstr{block: b, render: func() *Node { return f.formatVideo(b) }}
	case *PollBlock:
		return &renderInstr{block: b, render: func() *Node { return f.formatPoll(b) }}
	case *Unsupported:
		return &renderInstr{block: b, render: func() *Node { return f.formatUnsupported(b) }}
	}
	return nil
}

func calculatePad(block any) int {
	switch b := block.(type) {
	case *TextBlock:
		if len(b.Nest) == 0 {
			return 0
		}
		amount := len(b.Nest)
		for _, child := range b.Nest {
			if hasNested(child) {
				amount += calculatePad(child)
			}
		}
		return amount
	case *ListGrouping:
		if len(b.Group) == 0 {
			return 0
		}
		amount := len(b.Group) - 1
		for _, child := range b.Group {
			if hasNested(child) {
				amount += calculatePad(child)
			}
		}
		return amount
	}
	return 0
}

func hasNested(block any) bool {
	switch b := block.(type) {
	case *TextBlock:
		return len(b.Nest) > 0
	case *ListGrouping:
		return len(b.Group) > 0
	}
	return false
}

func (f *Formatter) pad() {
	for i := 0; i < f.currentContextPadding; i++ {
		f.renderInstructions = append(f.renderInstructions, nil)
	}
	f.currentContextPadding = 0
}

func (f *Formatter) formatList(block *ListGrouping) *Node {
	listTag := listElement(block.Type)
	for _, blk := range block.Group {
		listTag.Add(formatText(blk, f.urlHandler))
	}
	return listTag
}

func (f *Formatter) formatUnsupported(block *Unsupported) *Node {
	f.hasRenderError = true
	message := El("div", "class", "unsupported-content-block-message")
	message.Add(El("h1").Add(Txt(f.localizer.Translate("unsupported_block_header", nil))))
	message.Add(El("p").Add(Txt(f.localizer.Translate("unsupported_block_description", nil))))
	return El("div", "class", "unsupported-content-block").Add(message)
}

func (f *Formatter) formatImage(block *ImageBlock, rowLength int, overrideAspect *float64) *Node {
	figure := El("figure", "class", "image-block")
	figure.Add(formatImageContainer(block, rowLength, overrideAspect, f.urlHandler, f.localizer))

	if block.Caption != "" {
		figure.Add(El("figcaption", "class", "image-caption").Add(Txt(block.Caption)))
	}

	if attr := block.Attribution; attr != nil {
		switch attr.Kind {
		case AttrLink:
			figure.Add(formatLinkAttribution(*attr, f.urlHandler))
		case AttrPost:
			figure.Add(formatPostAttribution(*attr, f.urlHandler, f.localizer))
		case AttrBlog:
			figure.Add(formatBlogAttribution(*attr, f.urlHandler, f.localizer))
		case AttrApp:
			figure.Add(formatAppAttribution(*attr, f.urlHandler, f.localizer))
		default:
			figure.Add(formatUnsupportedAttribution(*attr, f.localizer))
		}
	}

	if f.ImageHook != nil {
		figure = f.ImageHook(f, block, rowLength, overrideAspect, figure)
	}
	return figure
}

func (f *Formatter) formatLink(block *LinkBlock) *Node {
	container := El("div", "class", "link-block")
	anchor := El("a", "href", f.urlHandler(block.URL), "class", "link-block-link")
	container.Add(anchor)

	if len(block.Poster) > 0 {
		poster := El("div", "class", "poster-container")
		anchor.Add(poster)

		srcset := strings.Join(createSrcset(block.Poster, f.urlHandler), ", ")
		alt := block.SiteName
		if alt == "" {
			alt = f.localizer.Translate("link_block_poster_alt_text", map[string]string{"site": block.URL})
		}
		poster.Add(El("img", "srcset", srcset, "alt", alt, "sizes", "(max-width: 540px) 100vh, 540px"))

		if block.Title != "" {
			poster.Add(El("div", "class", "link-block-title poster-overlay-text").
				Add(El("span").Add(Txt(block.Title))))
		}
	} else if block.Title != "" {
		anchor.Add(El("div", "class", "link-block-title").Add(El("span").Add(Txt(block.Title))))
	}

	descriptionContainer := El("div", "class", "link-block-description-container")
	anchor.Add(descriptionContainer)

	if block.Description != "" {
		descriptionContainer.Add(El("p", "class", "link-block-description").Add(Txt(block.Description)))
	}

	subtitlesDiv := El("div", "class", "link-block-subtitles")
	descriptionContainer.Add(subtitlesDiv)
	subtitles := El("span")
	subtitlesDiv.Add(subtitles)

	siteName := block.SiteName
	if siteName == "" {
		siteName = hostname(block.URL)
		if siteName == "" {
			siteName = block.URL
		}
	}

	if siteName != "" {
		subtitles.Add(El("span").Add(Txt(siteName)))
		if block.Author != "" {
			subtitles.Add(El("span", "class", "site-name-author-separator").Add(Txt("|")))
			subtitles.Add(El("span").Add(Txt(block.Author)))
		}
	} else {
		subtitles.Add(El("span").Add(Txt(block.Author)))
	}

	return container
}

func (f *Formatter) formatVideo(block *VideoBlock) *Node {
	var video *Node

	useNative := false
	mediaURL := ""
	host := ""
	if len(block.Media) > 0 {
		mediaURL = block.Media[0].URL
		host = hostname(mediaURL)
		useNative = block.Provider == "tumblr" || strings.HasSuffix(host, ".tumblr.com")
	}

	if useNative {
		if !strings.HasSuffix(host, ".tumblr.com") {
			return f.audiovisualFallback(
				block.URL, block.Media, block.Poster, block.Provider,
				f.localizer.Translate("error_link_block_fallback_native_video_player_non_tumblr_source", nil),
				f.localizer.Translate("video_link_block_fallback_description", nil), "")
		}

		width := block.Media[0].Width
		height := block.Media[0].Height

		video = El("video")
		video.Add(El("source", "src", f.urlHandler(mediaURL), "type", block.Media[0].Type))
		video.SetAttr("width", strconv.Itoa(width))
		video.SetAttr("height", strconv.Itoa(height))
		video.SetAttr("controls", "controls")
		if len(block.Poster) > 0 {
			video.SetAttr("poster", f.urlHandler(block.Poster[0].URL))
		}
	}

	if video == nil && !f.forbidExternalIframes {
		if block.EmbedIframe != nil {
			width := block.EmbedIframe.Width
			height := block.EmbedIframe.Height
			if block.Provider == "youtube" {
				height = 300
			}
			iframe := El("iframe",
				"src", block.EmbedIframe.URL,
				"width", strconv.Itoa(width),
				"height", strconv.Itoa(height),
				"scrolling", "no",
				"frameborder", "0")
			if block.Provider != "" {
				iframe.SetAttr("title", block.Provider)
			}
			video = iframe
		} else if block.EmbedHTML != "" {
			video = RawHTML(block.EmbedHTML)
		}
	}

	if video == nil {
		if f.forbidExternalIframes && (block.EmbedHTML != "" || block.EmbedURL != "" || block.EmbedIframe != nil) {
			return f.audiovisualFallback(
				block.URL, block.Media, block.Poster, block.Provider,
				f.localizer.Translate("link_block_fallback_embeds_are_disabled", nil),
				f.localizer.Translate("video_link_block_fallback_description", nil), "")
		}
		return f.audiovisualFallback(
			block.URL, block.Media, block.Poster, block.Provider,
			f.localizer.Translate("error_video_link_block_fallback_heading", nil),
			f.localizer.Translate("video_link_block_fallback_description", nil), "")
	}

	videoBlock := El("div", "class", "video-block")
	container := El("div", "class", "video-container")
	container.Add(video)
	videoBlock.Add(container)

	if f.VideoHook != nil {
		videoBlock = f.VideoHook(f, block, videoBlock)
	}
	return videoBlock
}

func (f *Formatter) formatAudio(block *AudioBlock) *Node {
	var audio *Node

	useNative := false
	mediaURL := ""
	host := ""
	if len(block.Media) > 0 {
		mediaURL = block.Media[0].URL
		host = hostname(mediaURL)
		useNative = block.Provider == "tumblr" || strings.HasSuffix(host, ".tumblr.com")
	}

	if useNative {
		if !strings.HasSuffix(host, ".tumblr.com") {
			return f.audiovisualFallback(
				block.URL, block.Media, block.Poster, block.Provider,
				f.localizer.Translate("error_link_block_fallback_native_audio_player_non_tumblr_source", nil),
				f.localizer.Translate("audio_link_block_fallback_description", nil), host)
		}

		container := El("section", "class", "ap-container")
		heading := El("header", "class", "ab-heading")
		metadata := El("div", "class", "ab-metadata")

		var metadataElements []*Node
		if block.Title != "" {
			metadataElements = append(metadataElements, El("h3", "class", "ab-title").Add(Txt(block.Title)))
		}
		if block.Artist != "" {
			metadataElements = append(metadataElements, El("h4", "class", "ab-artist").Add(Txt(block.Artist)))
		}
		if block.Album != "" {
			metadataElements = append(metadataElements, El("h4", "class", "ab-album").Add(Txt(block.Album)))
		}
		if len(metadataElements) > 0 {
			metadata.Add(metadataElements...)
			heading.Add(metadata)
		}

		if len(block.Poster) > 0 {
			alt := block.Title
			if alt == "" {
				alt = f.localizer.Translate("fallback_audio_block_thumbnail_alt_text", nil)
			}
			heading.Add(El("img",
				"src", f.urlHandler(block.Poster[0].URL),
				"srcset", strings.Join(createSrcset(block.Poster, f.urlHandler), ", "),
				"alt", alt,
				"sizes", "(max-width: 540px) 100vh, 540px",
				"class", "ab-poster"))
		}

		if len(metadataElements) > 0 || len(block.Poster) > 0 {
			container.Add(heading)
		}

		container.Add(El("audio", "controls", "controls").
			Add(El("source", "src", f.urlHandler(mediaURL), "type", block.Media[0].Type)))
		audio = container
	}

	if audio == nil && !f.forbidExternalIframes {
		if block.EmbedHTML != "" {
			audio = RawHTML(block.EmbedHTML)
		} else if block.EmbedURL != "" {
			audio = El("iframe", "src", block.EmbedURL, "scrolling", "no", "frameborder", "0")
		}
	}

	if audio == nil {
		if f.forbidExternalIframes && (block.EmbedHTML != "" || block.EmbedURL != "") {
			return f.audiovisualFallback(
				block.URL, block.Media, block.Poster, block.Provider,
				f.localizer.Translate("link_block_fallback_embeds_are_disabled", nil),
				f.localizer.Translate("audio_link_block_fallback_description", nil), "")
		}
		return f.audiovisualFallback(
			block.URL, block.Media, block.Poster, block.Provider,
			f.localizer.Translate("error_audio_link_block_fallback_heading", nil),
			f.localizer.Translate("audio_link_block_fallback_description", nil), "")
	}

	return El("div", "class", "audio-block").Add(audio)
}

func (f *Formatter) formatPoll(block *PollBlock) *Node {
	article := El("article", "class", "poll-block", "aria_label", "poll")
	header := El("header")
	header.Add(El("h3").Add(Txt(block.Question)))
	article.Add(header)

	body := El("section")
	choices := El("ul", "class", "poll-choices")

	for _, answer := range block.Answers {
		choice := El("li", "class", "poll-choice")
		choice.Add(El("h4", "class", "answer").Add(Txt(answer.Text)))

		if block.Votes != nil {
			result := block.Votes.Results[answer.ID]
			if result.IsWinner {
				choice.SetAttr("class", choice.Attrs["class"]+" poll-winner")
			}

			proportion := El("span", "class", "vote-proportion")
			if block.TotalVotes != 0 {
				width := round(float64(result.VoteCount)/float64(block.TotalVotes)*100, 3)
				proportion.SetAttr("style", "width: "+pyFloatStr(width)+"%;")
			}
			choice.Add(proportion)
			choice.Add(El("p", "class", "vote-count").
				Add(Txt(f.localizer.FormatDecimal("poll-choice-vote-count", float64(result.VoteCount)))))
		}
		choices.Add(choice)
	}
	body.Add(choices)
	article.Add(body)

	footer := El("footer")
	expiration := time.Unix(block.CreationTimestamp+block.ExpiresAfter, 0).UTC()
	now := time.Now().UTC()
	metadata := El("div", "class", "poll-metadata")

	if block.Votes != nil {
		metadata.Add(El("span").Add(Txt(f.localizer.TranslatePlural(
			"plural_poll_total_votes", block.TotalVotes,
			map[string]string{"votes": f.localizer.FormatDecimal("poll_votes", float64(block.TotalVotes))}))))
		metadata.Add(El("span", "class", "separator").Add(Txt("•")))
	}

	if expiration.After(now) {
		remaining := expiration.Sub(now)
		durationString := f.localizer.FormatDuration("poll_duration", remaining)
		timeTag := El("time", "datetime", buildDurationString(remaining)).Add(Txt(durationString))
		metadata.Add(El("span").Add(RawHTML(
			f.localizer.Translate("poll_remaining_time", map[string]string{"duration": timeTag.Render()}))))
	} else {
		human := f.localizer.FormatDatetime("poll_ended_on", expiration)
		formatted := expiration.Format("2006-01-02T15:04")
		timeTag := El("time", "datetime", formatted).Add(Txt(human))
		metadata.Add(El("span").Add(RawHTML(
			f.localizer.Translate("poll_ended_on", map[string]string{"ended_date": timeTag.Render()}))))
	}

	footer.Add(metadata)
	article.Add(footer)

	if now.After(expiration) {
		article.SetAttr("class", article.Attrs["class"]+" expired-poll")
	}
	if block.Votes != nil {
		article.SetAttr("class", article.Attrs["class"]+" populated")
	}

	if f.PollHook != nil {
		article = f.PollHook(f, block, article)
	}
	return article
}

func (f *Formatter) audiovisualFallback(aURL string, media, poster []MediaObject, provider, title, description, siteName string) *Node {
	url := aURL
	if url == "" {
		if len(media) > 0 {
			url = media[0].URL
		} else {
			return nil
		}
	}
	if url == "" {
		return nil
	}

	site := siteName
	if site == "" {
		site = provider
	}

	return f.formatLink(&LinkBlock{
		URL:         url,
		Title:       title,
		Description: description,
		Poster:      poster,
		SiteName:    site,
	})
}

// ---------------------------------------------------------------------------
// Top-level formatting
// ---------------------------------------------------------------------------

func (f *Formatter) createTruncation() *Node {
	return El("details", "class", "layout-truncated").Add(El("summary").Add(Txt("Read more")))
}

// Format renders all blocks and returns the post-body node.
func (f *Formatter) Format() *Node {
	f.post = El("div", "class", "post-body")

	for f.next() {
		instruction := f.prepare()
		f.renderInstructions = append(f.renderInstructions, instruction)
		f.pad()
	}

	if len(f.layout) > 0 {
		f.applyLayout()
	} else {
		for _, instruction := range f.renderInstructions {
			if instruction == nil {
				continue
			}
			f.post.Add(instruction.render())
		}
	}

	return f.post
}

type imageRowItem struct {
	block *ImageBlock
	orig  MediaObject
}

func instructionAt(list []*renderInstr, i int) *renderInstr {
	if i < 0 || i >= len(list) {
		return nil
	}
	return list[i]
}

func originalMedia(block *ImageBlock) MediaObject {
	for _, m := range block.Media {
		if m.HasOriginalDimensions {
			return m
		}
	}
	if len(block.Media) > 0 {
		return block.Media[0]
	}
	return MediaObject{}
}

func (f *Formatter) applyLayout() {
	blocksInLayouts := map[int]bool{}
	rowAttachment := f.post
	lastBlockIndex := 0

	for _, layout := range f.layout {
		switch lay := layout.(type) {
		case *Rows:
			for _, row := range lay.Rows {
				for _, r := range row.Ranges {
					blocksInLayouts[r] = true
				}

				var rowItems []any
				hasImage := false

				for _, blockIndex := range row.Ranges {
					lastBlockIndex = blockIndex
					instruction := instructionAt(f.renderInstructions, blockIndex)
					if instruction == nil {
						continue
					}
					if img, ok := instruction.block.(*ImageBlock); ok {
						hasImage = true
						rowItems = append(rowItems, imageRowItem{block: img, orig: originalMedia(img)})
					} else {
						rowItems = append(rowItems, instruction.render())
					}
				}

				if len(rowItems) == 0 {
					continue
				}

				if hasImage {
					var imageIndices []int
					var ratios []float64
					for i, item := range rowItems {
						if iri, ok := item.(imageRowItem); ok {
							imageIndices = append(imageIndices, i)
							ratios = append(ratios, round(float64(iri.orig.Width)/float64(iri.orig.Height), 4))
						}
					}
					aspectRatio := maxFloat(ratios)
					for _, i := range imageIndices {
						iri := rowItems[i].(imageRowItem)
						rowItems[i] = f.formatImage(iri.block, len(imageIndices), &aspectRatio)
					}
				}

				rowTag := El("div", "class", "layout-row")

				if f.truncate && lay.TruncateAfter != nil && lastBlockIndex > *lay.TruncateAfter {
					if rowAttachment == f.post {
						rowAttachment = f.createTruncation()
						f.post.Add(rowAttachment)
					}
				}

				rowAttachment.Add(rowTag)
				for _, item := range rowItems {
					if node, ok := item.(*Node); ok {
						rowTag.Add(node)
					}
				}
			}
		case *AskLayout:
			var items []*Node
			for _, index := range lay.Ranges {
				blocksInLayouts[index] = true
				instruction := instructionAt(f.renderInstructions, index)
				if instruction == nil {
					continue
				}
				items = append(items, instruction.render())
			}
			f.post.Add(El("div", "class", "layout-ask").
				Add(formatAsk(lay.Attribution, f.urlHandler, f.localizer, items...)))
		}
	}

	// Edge case: only an "ask" layout was given.
	if len(f.layout) == 1 {
		if _, ok := f.layout[0].(*AskLayout); ok {
			for index, instruction := range f.renderInstructions {
				if blocksInLayouts[index] || instruction == nil {
					continue
				}
				f.post.Add(El("div", "class", "layout-row").Add(instruction.render()))
			}
		}
	}
}

func maxFloat(values []float64) float64 {
	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

// buildDurationString renders an ISO 8601 duration string, mirroring
// npf_renderer.helpers.build_duration_string.
func buildDurationString(d time.Duration) string {
	secs := int64(d / time.Second)
	days := secs / 86400
	rem := secs % 86400
	hours := rem / 3600
	minutes := (rem % 3600) / 60
	seconds := rem % 60

	out := "P"
	if days != 0 {
		out += fmt.Sprintf("%dD", days)
	}
	if hours != 0 || minutes != 0 || seconds != 0 {
		out += "T"
		if hours != 0 {
			out += fmt.Sprintf("%dH", hours)
		}
		if minutes != 0 {
			out += fmt.Sprintf("%dM", minutes)
		}
		if seconds != 0 {
			out += fmt.Sprintf("%dS", seconds)
		}
	}
	return out
}
