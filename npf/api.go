package npf

// Options configure NPF formatting.
type Options struct {
	// URLHandler rewrites every URL emitted into the HTML.
	URLHandler func(string) string
	// Localizer supplies translated strings and value formatting.
	Localizer Localizer
	// ForbidExternalIframes suppresses external embed iframes.
	ForbidExternalIframes bool
	// Truncate cuts the post at the layout's truncation point.
	Truncate bool
	// PollCallback fetches poll results, if any.
	PollCallback PollCallback

	// Hooks for openblur-specific augmentation.
	ImageHook func(f *Formatter, block *ImageBlock, rowLength int, overrideAspect *float64, node *Node) *Node
	VideoHook func(f *Formatter, block *VideoBlock, node *Node) *Node
	PollHook  func(f *Formatter, block *PollBlock, node *Node) *Node

	// BlogName and PostID are exposed to PollHook for its no-JS fallback.
	BlogName string
	PostID   string
}

// NewFormatter builds a Formatter from already-parsed content and layout.
func NewFormatter(content, layout []any, opts Options) *Formatter {
	urlHandler := opts.URLHandler
	if urlHandler == nil {
		urlHandler = func(u string) string { return u }
	}
	loc := opts.Localizer
	if loc == nil {
		loc = DefaultLocalizer{}
	}
	return &Formatter{
		content:               content,
		layout:                layout,
		localizer:             loc,
		urlHandler:            urlHandler,
		forbidExternalIframes: opts.ForbidExternalIframes,
		truncate:              opts.Truncate,
		ImageHook:             opts.ImageHook,
		VideoHook:             opts.VideoHook,
		PollHook:              opts.PollHook,
		BlogName:              opts.BlogName,
		PostID:                opts.PostID,
	}
}

// Format parses and renders NPF content. It returns whether the rendered
// result contains errors, and the rendered HTML.
func Format(content []any, layout []any, opts Options) (bool, string) {
	parsed, err := NewParser(content, opts.PollCallback).Parse()
	if err != nil {
		return true, `<div class="post-body"></div>`
	}

	var parsedLayout []any
	if len(layout) > 0 {
		parsedLayout = NewLayoutParser(layout).Parse()
	}

	formatter := NewFormatter(parsed, parsedLayout, opts)
	node := formatter.Format()
	return formatter.HasRenderError(), node.Render()
}
