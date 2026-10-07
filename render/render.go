// Package render turns Tumblr NPF posts into hyperblur-flavoured HTML.
//
// It wraps the npf package with hyperblur's customisations: media proxying,
// download buttons, image alt-text widgets, and poll data attributes.
package render

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/dan1165/hyperblur/npf"
)

// downloadIcon is overlaid on every downloadable media element.
const downloadIcon = `<svg class="icon" xmlns="http://www.w3.org/2000/svg" height="20" width="20" viewBox="0 -960 960 960"><path d="M480-320 280-520l56-58 104 104v-326h80v326l104-104 56 58-200 200ZM240-160q-33 0-56.5-23.5T160-240v-120h80v120h480v-120h80v120q0 33-23.5 56.5T720-160H240Z"/></svg>`

// RenderError describes a post-render failure, mirroring the tuple returned by
// hyperblur's create_user_friendly_error_message.
type RenderError struct {
	Name    string
	Message string
}

// Params configure post rendering.
type Params struct {
	// BlogName and PostID are used for the poll no-JS fallback link.
	BlogName string
	PostID   string
	// PollCallback fetches poll results.
	PollCallback npf.PollCallback
}

// FormatNPF renders NPF content to hyperblur HTML. It returns a RenderError and
// an error placeholder body when rendering fails, or nil and the post body.
func FormatNPF(content, layout []any, p Params) (*RenderError, string) {
	opts := npf.Options{
		URLHandler:   URLHandler,
		Localizer:    Localizer{},
		PollCallback: p.PollCallback,
		BlogName:     p.BlogName,
		PostID:       p.PostID,
		ImageHook:    imageHook,
		VideoHook:    videoHook,
		PollHook:     pollHook,
	}

	hasError, html := npf.Format(content, layout, opts)
	if hasError {
		return &RenderError{
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
		"href", mediaURL+"?download=1",
		"download", "",
		"class", "media-download",
		"title", "Download",
		"aria_label", "Download media",
	).Add(npf.RawHTML(downloadIcon)))
}

// URLHandler rewrites Tumblr URLs found in posts into hyperblur's own
// privacy-friendly paths.
func URLHandler(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	hostname := strings.ToLower(u.Hostname())
	path := u.EscapedPath()
	if path == "" {
		path = u.Path
	}

	// Redirect links can hold malformed URLs (e.g. https://href.li/?http://);
	// recurse into the wrapped URL.
	if strings.HasSuffix(hostname, "href.li") {
		return URLHandler(u.RawQuery)
	}
	if strings.HasSuffix(hostname, "t.umblr.com") {
		if values, err := url.ParseQuery(u.RawQuery); err == nil {
			if redirect := values.Get("z"); redirect != "" {
				return URLHandler(redirect)
			}
		}
	}

	if hostname != "" && strings.HasSuffix(hostname, "tumblr.com") {
		if strings.HasSuffix(hostname, ".media.tumblr.com") {
			subDomains := strings.Split(hostname, ".")
			if len(subDomains) > 1 && subDomains[1] == "media" {
				return "/tblr/media/" + subDomains[0] + path
			}
			if len(subDomains) > 2 && subDomains[0] == "www" && subDomains[2] == "media" {
				return "/tblr/media/" + subDomains[1] + path
			}
		}

		switch {
		case strings.HasSuffix(hostname, "assets.tumblr.com"):
			return "/tblr/assets" + path
		case strings.HasSuffix(hostname, "static.tumblr.com"):
			return "/tblr/static" + path
		case strings.HasPrefix(hostname, "a."):
			return "/tblr/a" + path
		default:
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
	}

	return u.String()
}
