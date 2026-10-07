package npf

import "time"

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
