package npf

import (
	"fmt"
	"net/url"
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

	localizer  Localizer
	urlHandler func(string) string

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

	if video == nil {
		if block.EmbedHTML != "" || block.EmbedURL != "" || block.EmbedIframe != nil {
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

	if audio == nil {
		if block.EmbedHTML != "" || block.EmbedURL != "" {
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
			result := block.Votes[answer.ID]
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

				f.post.Add(rowTag)
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

// pyFloatStr renders a float the way Python's str() would, so a whole number
// keeps a trailing ".0".
func pyFloatStr(f float64) string {
	s := fmt.Sprintf("%v", f)
	if !strings.ContainsAny(s, ".eE") {
		s += ".0"
	}
	return s
}

// createSrcset renders media objects into srcset data.
func createSrcset(media []MediaObject, urlHandler func(string) string) []string {
	out := make([]string, 0, len(media))
	for _, m := range media {
		out = append(out, fmt.Sprintf("%s %dw", urlHandler(m.URL), m.Width))
	}
	return out
}

// formatImageContainer renders the inner <div class="image-container"> of an
// image block, mirroring npf_renderer.format.image.format_image.
func formatImageContainer(block *ImageBlock, rowLength int, overrideAspect *float64, urlHandler func(string) string, loc Localizer) *Node {
	containerStyle := ""
	imageStyle := ""
	var processed []MediaObject
	var original *MediaObject
	withAspect := true

	for i := range block.Media {
		m := block.Media[i]
		if m.Cropped {
			continue
		}
		processed = append(processed, m)
		if m.HasOriginalDimensions {
			orig := processed[len(processed)-1]
			original = &orig
		}
	}

	if original == nil {
		switch {
		case len(processed) > 0:
			orig := processed[0]
			original = &orig
		case len(block.Media) > 0:
			orig := block.Media[0]
			original = &orig
			imageStyle = fmt.Sprintf("width: %dpx; height: %dpx;", original.Width, original.Height)
			withAspect = false
		default:
			return El("div", "class", "image-container")
		}
	}

	if withAspect {
		if overrideAspect != nil {
			containerStyle = "aspect-ratio: " + pyFloatStr(*overrideAspect) + ";"
		} else {
			ratio := round(float64(original.Width)/float64(original.Height), 4)
			containerStyle = "aspect-ratio: " + pyFloatStr(ratio) + ";"
		}
	}

	container := El("div", "class", "image-container")
	if containerStyle != "" {
		container.SetAttr("style", containerStyle)
	}

	alt := block.AltText
	if alt == "" {
		alt = loc.Translate("generic_image_alt_text", nil)
	}

	img := El("img")
	img.SetAttr("src", urlHandler(original.URL))
	img.SetAttr("srcset", strings.Join(createSrcset(processed, urlHandler), ", "))
	img.SetAttr("class", "image")
	img.SetAttr("loading", "lazy")
	img.SetAttr("alt", alt)
	img.SetAttr("sizes", fmt.Sprintf("(max-width: 540px) %dvh, %dpx", 100/rowLength, 540/rowLength))
	if imageStyle != "" {
		img.SetAttr("style", imageStyle)
	}
	container.Add(img)
	return container
}

// round rounds f to n decimal places the way Python's round() does
// (half-to-even), then returns it as a float.
func round(f float64, n int) float64 {
	s := strconv.FormatFloat(f, 'f', n, 64)
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func hostname(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

func formatLinkAttribution(attr Attribution, urlHandler func(string) string) *Node {
	return El("div", "class", "link-attribution").
		Add(El("a", "href", urlHandler(attr.URL)).Add(Txt(hostname(attr.URL))))
}

func formatPostAttribution(attr Attribution, urlHandler func(string) string, loc Localizer) *Node {
	name := El("b").Add(Txt(attr.Blog.Name)).Render()
	inner := loc.Translate("post_attribution", map[string]string{"author": name})
	return El("div", "class", "post-attribution").
		Add(El("a", "href", urlHandler(attr.URL)).Add(RawHTML(inner)))
}

func formatBlogAttribution(attr Attribution, urlHandler func(string) string, loc Localizer) *Node {
	name := attr.Blog.Name
	if name == "" {
		name = "Anonymous"
	}
	author := El("b").Add(Txt(name)).Render()
	inner := loc.Translate("blog_attribution", map[string]string{"author": author})
	return El("div", "class", "blog-attribution").
		Add(El("a", "href", urlHandler(attr.URL)).Add(RawHTML(inner)))
}

func formatAppAttribution(attr Attribution, urlHandler func(string) string, loc Localizer) *Node {
	app := El("b").Add(Txt(attr.AppName)).Render()
	inner := loc.Translate("app_attribution", map[string]string{"platform": app})
	return El("div", "class", "post-attribution").
		Add(El("a", "href", urlHandler(attr.URL)).Add(RawHTML(inner)))
}

func formatUnsupportedAttribution(attr Attribution, loc Localizer) *Node {
	inner := loc.Translate("unsupported_attribution", map[string]string{"attributee": attr.TypeStr})
	return El("div", "class", "unknown-attribution").Add(El("p").Add(Txt(inner)))
}

// formatAsk renders an "ask" layout with the given pre-rendered contents.
func formatAsk(attr *BlogAttribution, urlHandler func(string) string, loc Localizer, contents ...*Node) *Node {
	var askerAttribution *Node
	var askerAvatar *Node

	if attr == nil {
		name := El("strong", "class", "asker-name").Add(Txt(loc.Translate("asker_with_no_attribution", nil)))
		asked := loc.Translate("asker_and_ask_verb", map[string]string{"name": name.Render()})
		askerAttribution = El("p", "class", "asker").Add(RawHTML(asked))
		askerAvatar = El("img",
			"src", urlHandler("https://assets.tumblr.com/images/anonymous_avatar_96.gif"),
			"loading", "lazy",
			"class", "avatar asker-avatar image")
	} else {
		nameHTML := El("a",
			"href", urlHandler(fmt.Sprintf("https://%s.tumblr.com/", attr.Name)),
			"class", "asker-attribution").
			Add(El("strong", "class", "asker-name").Add(Txt(attr.Name))).Render()
		asked := loc.Translate("asker_and_ask_verb", map[string]string{"name": nameHTML})
		askerAttribution = El("p", "class", "asker").Add(RawHTML(asked))

		if len(attr.Avatar) > 0 {
			askerAvatar = El("img",
				"src", urlHandler(attr.Avatar[0].URL),
				"loading", "lazy",
				"class", "avatar asker-avatar image")
		}
	}

	ask := El("div", "class", "ask")
	body := El("div", "class", "ask-body")
	header := El("div", "class", "ask-header")
	header.Add(askerAttribution)
	content := El("div", "class", "ask-content")
	content.Add(contents...)
	body.Add(header, content)
	ask.Add(body)
	if askerAvatar != nil {
		ask.Add(askerAvatar)
	}
	return ask
}

// Localizer supplies the human-readable strings and value formatting used
// while rendering.
type Localizer interface {
	// Translate returns the (optionally substituted) string for key.
	Translate(key string, subst map[string]string) string
	// TranslatePlural returns the correct plural form for number.
	TranslatePlural(key string, number int, subst map[string]string) string
	FormatDuration(key string, d time.Duration) string
	FormatDatetime(key string, t time.Time) string
	FormatDecimal(key string, n float64) string
}

// Options configure NPF formatting.
type Options struct {
	// URLHandler rewrites every URL emitted into the HTML.
	URLHandler func(string) string
	// Localizer supplies translated strings and value formatting.
	Localizer Localizer
	// PollCallback fetches poll results, if any.
	PollCallback PollCallback

	// Hooks for hyperblur-specific augmentation.
	ImageHook func(f *Formatter, block *ImageBlock, rowLength int, overrideAspect *float64, node *Node) *Node
	VideoHook func(f *Formatter, block *VideoBlock, node *Node) *Node
	PollHook  func(f *Formatter, block *PollBlock, node *Node) *Node

	// BlogName and PostID are exposed to PollHook for its no-JS fallback.
	BlogName string
	PostID   string
}

// newFormatter builds a Formatter from already-parsed content and layout.
func newFormatter(content, layout []any, opts Options) *Formatter {
	urlHandler := opts.URLHandler
	if urlHandler == nil {
		urlHandler = func(u string) string { return u }
	}
	return &Formatter{
		content:    content,
		layout:     layout,
		localizer:  opts.Localizer,
		urlHandler: urlHandler,
		ImageHook:  opts.ImageHook,
		VideoHook:  opts.VideoHook,
		PollHook:   opts.PollHook,
		BlogName:   opts.BlogName,
		PostID:     opts.PostID,
	}
}

// Format parses and renders NPF content. It returns whether the rendered
// result contains errors, and the rendered HTML.
func Format(content []any, layout []any, opts Options) (bool, string) {
	parsed, err := newParser(content, opts.PollCallback).Parse()
	if err != nil {
		return true, `<div class="post-body"></div>`
	}

	var parsedLayout []any
	if len(layout) > 0 {
		parsedLayout = newLayoutParser(layout).Parse()
	}

	formatter := newFormatter(parsed, parsedLayout, opts)
	node := formatter.Format()
	return formatter.HasRenderError(), node.Render()
}
