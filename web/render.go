package web

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/dan1165/hyperblur/npf"
)

// downloadIcon overlays every downloadable media element.
const downloadIcon = `<svg class="icon" xmlns="http://www.w3.org/2000/svg" height="20" width="20" viewBox="0 -960 960 960"><path d="M480-320 280-520l56-58 104 104v-326h80v326l104-104 56 58-200 200ZM240-160q-33 0-56.5-23.5T160-240v-120h80v120h480v-120h80v120q0 33-23.5 56.5T720-160H240Z"/></svg>`

// renderError describes post-render failure, mirroring hyperblur's
// create_user_friendly_error_message.
type renderError struct {
	Name    string
	Message string
}

// params configure post render.
type params struct {
	// BlogName and PostID feed poll no-JS fallback link.
	BlogName string
	PostID   string
	// PollCallback fetches poll results.
	PollCallback npf.PollCallback
	// Ctx bounds upstream work done while rendering; nil means Background.
	Ctx context.Context
	// EagerBudget is the page's shared eager-image budget, shared with the
	// renderer so one post cannot spend more than the page allows.
	EagerBudget *int
}

// formatNPF renders NPF content to hyperblur HTML. Failure returns renderError
// and error placeholder body; success returns nil and post body.
func formatNPF(content, layout []any, p params) (*renderError, string) {
	opts := npf.Options{
		URLHandler:   urlHandler,
		Localizer:    localizer{},
		PollCallback: p.PollCallback,
		BlogName:     p.BlogName,
		PostID:       p.PostID,
		ImageHook:    imageHook,
		VideoHook:    videoHook,
		PollHook:     pollHook,
		EagerBudget:  p.EagerBudget,
	}

	hasError, html := npf.Format(content, layout, opts)
	if hasError {
		return &renderError{
			Name:    "RenderErrorDisclaimerError",
			Message: "Rendered post contains errors",
		}, `<div class="post-body has-error"></div>`
	}
	return nil, html
}

func imageHook(f *npf.Formatter, block *npf.ImageBlock, _ int, _ *float64, node *npf.Node) *npf.Node {
	image := node.Find("img")
	container := node.FindClass("image-container")
	if image == nil || container == nil {
		return node
	}

	if block.AltText != "" && block.AltText != "image" {
		details := npf.El("details").
			Add(npf.El("summary", "title", block.AltText).Add(npf.Txt("ALT"))).
			Add(npf.El("p").Add(npf.Txt(block.AltText)))
		container.Add(npf.El("div", "class", "img-alt-text").Add(details))
	}

	src := image.Attrs["src"]
	for i, child := range container.Kids {
		if child == image {
			container.Kids[i] = npf.El("a", "href", src).Add(image)
			break
		}
	}

	addDownloadButton(container, src)
	return node
}

func videoHook(f *npf.Formatter, block *npf.VideoBlock, node *npf.Node) *npf.Node {
	if node.Find("video") == nil || len(block.Media) == 0 {
		return node
	}
	container := node.FindClass("video-container")
	if container == nil {
		return node
	}
	addDownloadButton(container, f.URLHandler(block.Media[0].URL))
	return node
}

func pollHook(f *npf.Formatter, block *npf.PollBlock, node *npf.Node) *npf.Node {
	node.SetAttr("data-poll-id", block.PollID)

	if choices := node.FindClass("poll-choices"); choices != nil {
		for i, answer := range block.Answers {
			if i < len(choices.Kids) {
				choices.Kids[i].SetAttr("data-answer-id", answer.ID)
			}
		}
	}

	if block.Votes == nil && f.BlogName != "" && f.PostID != "" {
		if footer := node.Find("footer"); footer != nil {
			href := fmt.Sprintf("/%s/%s?fetch_polls=true", f.BlogName, f.PostID)
			fallback := npf.El("noscript").
				Add(npf.El("a", "href", href, "class", "toggle-poll-results").Add(npf.Txt("See Results")))
			footer.Prepend(fallback)
		}
	}
	return node
}

func addDownloadButton(container *npf.Node, mediaURL string) {
	if mediaURL == "" {
		return
	}
	container.Add(npf.El("a",
		"href", downloadURL(mediaURL),
		"download", "",
		"class", "media-download",
		"title", "Download",
		"aria_label", "Download media",
	).Add(npf.RawHTML(downloadIcon)))
}

// urlHandler rewrites Tumblr URLs in posts to hyperblur's privacy-friendly paths.
func urlHandler(rawURL string) string { return unwrapURL(rawURL, 0) }

// maxUnwrapDepth caps recursion into wrapped redirect links, so a post cannot
// nest href.li/t.umblr.com redirects deeply enough to exhaust the stack.
const maxUnwrapDepth = 4

func unwrapURL(rawURL string, depth int) string {
	if depth > maxUnwrapDepth {
		return rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	hostname := strings.ToLower(u.Hostname())
	path := u.EscapedPath()
	if path == "" {
		path = u.Path
	}

	// Redirect links can hold malformed URLs (e.g. https://href.li/?http://).
	// Recurse into wrapped URL.
	if strings.HasSuffix(hostname, "href.li") {
		return unwrapURL(u.RawQuery, depth+1)
	}
	if strings.HasSuffix(hostname, "t.umblr.com") {
		if values, err := url.ParseQuery(u.RawQuery); err == nil {
			if redirect := values.Get("z"); redirect != "" {
				return unwrapURL(redirect, depth+1)
			}
		}
	}

	if hostname != "" && strings.HasSuffix(hostname, "tumblr.com") {
		// Media, asset and static hosts keep their absolute CDN URL: display
		// media loads straight from Tumblr. Only downloads come through
		// hyperblur, see downloadURL.
		if strings.Contains(hostname, ".media.tumblr.com") ||
			strings.HasSuffix(hostname, "assets.tumblr.com") ||
			strings.HasSuffix(hostname, "static.tumblr.com") ||
			strings.HasPrefix(hostname, "a.") {
			return u.String()
		}

		subDomains := strings.Split(hostname, ".")
		potentialBlogName := subDomains[0]
		if subDomains[0] == "www" && len(subDomains) > 1 {
			potentialBlogName = subDomains[1]
		}

		if potentialBlogName != "tumblr" {
			if strings.HasPrefix(path, "/post") {
				return "/" + potentialBlogName + path[5:]
			}
			return "/" + potentialBlogName + path
		}
		return path
	}

	return u.String()
}

// downloadPath maps a Tumblr media URL to hyperblur's /download route, or ""
// when the URL is not media the download route can fetch.
func downloadPath(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	hostname := strings.ToLower(u.Hostname())
	path := u.EscapedPath()
	if path == "" {
		path = u.Path
	}
	if hostname == "" || !strings.HasSuffix(hostname, ".media.tumblr.com") {
		return ""
	}

	subDomains := strings.Split(hostname, ".")
	if len(subDomains) > 1 && subDomains[1] == "media" {
		return withQuery("/download/"+subDomains[0]+path, u.RawQuery)
	}
	if len(subDomains) > 2 && subDomains[0] == "www" && subDomains[2] == "media" {
		return withQuery("/download/"+subDomains[1]+path, u.RawQuery)
	}
	return ""
}

// withQuery appends rawQuery verbatim, so signed CDN URLs keep working.
func withQuery(path, rawQuery string) string {
	if rawQuery == "" {
		return path
	}
	return path + "?" + rawQuery
}

// downloadURL points a media link at hyperblur's download route, which streams
// the file and names it. Non-Tumblr URLs get a best-effort download flag.
func downloadURL(rawURL string) string {
	if p := downloadPath(rawURL); p != "" {
		return p
	}
	return setQueryParam(rawURL, "download", "1")
}

// setQueryParam returns rawURL with key=value set in its query, replacing any
// existing value.
func setQueryParam(rawURL, key, value string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	query := u.Query()
	query.Set(key, value)
	u.RawQuery = query.Encode()
	return u.String()
}

// localizer provides hyperblur's English strings and formatting to
// npf.Localizer.
type localizer struct{}

// Translate implements npf.Localizer.
func (localizer) Translate(key string, subst map[string]string) string {
	return substitute(npfString(key), subst)
}

// TranslatePlural implements npf.Localizer.
func (localizer) TranslatePlural(key string, number int, subst map[string]string) string {
	if strings.TrimPrefix(key, "plural_") == "poll_total_votes" {
		text := "{votes} votes"
		if number == 1 {
			text = "{votes} vote"
		}
		return substitute(text, subst)
	}
	return substitute(npfString(strings.TrimPrefix(key, "plural_")), subst)
}

func substitute(text string, subst map[string]string) string {
	for k, v := range subst {
		text = strings.ReplaceAll(text, "{"+k+"}", v)
	}
	return text
}

// npfString holds strings NPF formatter asks localizer for.
func npfString(key string) string {
	switch key {
	case "asker_with_no_attribution":
		return "Anonymous"
	case "asker_and_ask_verb":
		return "{name} asked"
	case "unsupported_block_header":
		return "Unsupported NPF block"
	case "unsupported_block_description":
		return "Placeholder for the unsupported \"{block}\" type NPF block Please report me over at https://github.com/syeopite/npf-renderer"
	case "generic_image_alt_text":
		return "image"
	case "link_block_poster_alt_text":
		return "Preview image for \"{site}\""
	case "link_block_fallback_embeds_are_disabled":
		return "Embeds are disabled"
	case "error_video_link_block_fallback_heading":
		return "Error: unable to render video block"
	case "video_link_block_fallback_description":
		return "Please click me to watch on the original site"
	case "error_link_block_fallback_native_video_player_non_tumblr_source":
		return "Error: non-tumblr source for video player"
	case "fallback_audio_block_thumbnail_alt_text":
		return "Album art"
	case "error_audio_link_block_fallback_heading":
		return "Error: unable to render audio block"
	case "audio_link_block_fallback_description":
		return "Please click me to listen on the original site"
	case "error_link_block_fallback_native_audio_player_non_tumblr_source":
		return "Error: non-tumblr source for audio player"
	case "poll_remaining_time":
		return "{duration} remaining"
	case "poll_ended_on":
		return "Ended on: {ended_date}"
	case "post_attribution":
		return "From {author}"
	case "blog_attribution":
		return "Created by {author}"
	case "app_attribution":
		return "View on {platform}"
	case "unsupported_attribution":
		return "Attributed via an unsupported (\"{attributee}\") attribution type. Please report this over at https://github.com/syeopite/npf-renderer"
	}
	return key
}

// FormatDuration implements npf.Localizer.
//
// ponytail: approximates babel's format_timedelta(threshold=1.1) with largest
// unit whose value >= 1.1. Swap golang.org/x/text for exact babel wording.
func (localizer) FormatDuration(key string, d time.Duration) string {
	units := []struct {
		seconds float64
		name    string
	}{
		{365 * 86400, "year"},
		{30 * 86400, "month"},
		{7 * 86400, "week"},
		{86400, "day"},
		{3600, "hour"},
		{60, "minute"},
		{1, "second"},
	}
	seconds := d.Seconds()
	for _, u := range units {
		value := seconds / u.seconds
		if value >= 1.1 {
			n := int(math.Round(value))
			if n == 1 {
				return "1 " + u.name
			}
			return fmt.Sprintf("%d %ss", n, u.name)
		}
	}
	return "0 seconds"
}

// FormatDatetime implements npf.Localizer. babel en_US "short": M/d/yy, h:mm a,
// narrow no-break space (U+202F) before AM/PM.
func (localizer) FormatDatetime(key string, t time.Time) string {
	return t.Format("1/2/06, 3:04") + "\u202f" + t.Format("PM")
}

// FormatDecimal implements npf.Localizer with en_US thousands grouping.
func (localizer) FormatDecimal(key string, n float64) string {
	negative := n < 0
	if negative {
		n = -n
	}

	s := strconv.FormatFloat(n, 'f', -1, 64)
	intPart := s
	frac := ""
	if i := strings.IndexByte(s, '.'); i >= 0 {
		intPart, frac = s[:i], s[i:]
	}

	var b strings.Builder
	for i, ch := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(ch)
	}
	if negative {
		return "-" + b.String() + frac
	}
	return b.String() + frac
}
