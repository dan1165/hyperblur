// Package render turns Tumblr NPF posts into openblur-flavoured HTML.
//
// It wraps the npf package with openblur's customisations: media proxying,
// download buttons, image alt-text widgets, and poll data attributes, matching
// helpers/ext_npf_renderer.py.
package render

import (
	"fmt"

	"github.com/dan1165/openblur/helpers"
	"github.com/dan1165/openblur/npf"
)

// downloadIcon is overlaid on every downloadable media element.
const downloadIcon = `<svg class="icon" xmlns="http://www.w3.org/2000/svg" height="20" width="20" viewBox="0 -960 960 960"><path d="M480-320 280-520l56-58 104 104v-326h80v326l104-104 56 58-200 200ZM240-160q-33 0-56.5-23.5T160-240v-120h80v120h480v-120h80v120q0 33-23.5 56.5T720-160H240Z"/></svg>`

// RenderError describes a post-render failure, mirroring the tuple returned by
// openblur's create_user_friendly_error_message.
type RenderError struct {
	Name    string
	Message string
	Context string
}

// Params configure post rendering.
type Params struct {
	// ExpandPosts disables layout truncation when true.
	ExpandPosts bool
	// BlogName and PostID are used for the poll no-JS fallback link.
	BlogName string
	PostID   string
	// URLHandler rewrites URLs; defaults to helpers.URLHandler.
	URLHandler func(string) string
	// PollCallback fetches poll results.
	PollCallback npf.PollCallback
}

// FormatNPF renders NPF content to openblur HTML. It returns a RenderError and
// an error placeholder body when rendering fails, or nil and the post body.
func FormatNPF(content, layout []any, p Params) (*RenderError, string) {
	urlHandler := p.URLHandler
	if urlHandler == nil {
		urlHandler = helpers.URLHandler
	}

	opts := npf.Options{
		URLHandler:            urlHandler,
		Localizer:             Localizer{},
		ForbidExternalIframes: true,
		Truncate:              !p.ExpandPosts,
		PollCallback:          p.PollCallback,
		BlogName:              p.BlogName,
		PostID:                p.PostID,
		ImageHook:             imageHook,
		VideoHook:             videoHook,
		PollHook:              pollHook,
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
