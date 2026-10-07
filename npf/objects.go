package npf

// This file defines the parsed NPF object model, mirroring
// npf_renderer.objects.

// Subtype is a text block subtype.
type Subtype int

const (
	SubtypeNone Subtype = iota
	Heading1
	Heading2
	Quirky
	Quote
	Indented
	Chat
	OrderedListItem
	UnorderedListItem
)

func isListSubtype(s Subtype) bool {
	return s == OrderedListItem || s == UnorderedListItem
}

// FMTType is an inline formatting type.
type FMTType int

const (
	FMTBold FMTType = iota
	FMTItalic
	FMTStrikethrough
	FMTSmall
	FMTLink
	FMTMention
	FMTColor
)

// Instruction is a single inline formatting operation. Only the fields
// relevant to its Type are populated.
type Instruction struct {
	Type     FMTType
	URL      string
	BlogName string
	BlogURL  string
	BlogUUID string
	Hex      string
}

// StyleInterval is a run of text sharing a set of inline formatting
// instructions.
type StyleInterval struct {
	Start        int
	End          int
	Instructions []Instruction
}

// TextBlock is a parsed NPF text content block.
type TextBlock struct {
	Text             string
	Subtype          Subtype
	Nest             []any // *TextBlock or *ListGrouping
	InlineFormatting []StyleInterval
}

// ListGrouping groups adjacent list-item text blocks into one <ul>/<ol>.
type ListGrouping struct {
	Type  Subtype
	Group []*TextBlock
}

// MediaObject is an NPF media object.
type MediaObject struct {
	URL                   string
	Width                 int
	Height                int
	Type                  string
	Cropped               bool
	HasOriginalDimensions bool
}

// AttributionKind discriminates the attribution union.
type AttributionKind int

const (
	AttrNone AttributionKind = iota
	AttrPost
	AttrLink
	AttrBlog
	AttrApp
	AttrUnsupported
)

// BlogAttribution describes a blog.
type BlogAttribution struct {
	URL    string
	Name   string
	Avatar []MediaObject
}

// Attribution is the parsed attribution for a media block. Kind selects which
// fields are meaningful.
type Attribution struct {
	Kind    AttributionKind
	URL     string
	Blog    BlogAttribution
	AppName string
	TypeStr string
}

// ImageBlock is a parsed NPF image content block.
type ImageBlock struct {
	Media       []MediaObject
	AltText     string
	Caption     string
	Attribution *Attribution
}

// LinkBlock is a parsed NPF link content block.
type LinkBlock struct {
	URL         string
	Title       string
	Description string
	Author      string
	SiteName    string
	Poster      []MediaObject
}

// EmbedIframe is embed iframe metadata.
type EmbedIframe struct {
	URL    string
	Width  int
	Height int
}

// AudioBlock is a parsed NPF audio content block.
type AudioBlock struct {
	URL       string
	Provider  string
	Media     []MediaObject
	Title     string
	Artist    string
	Album     string
	Poster    []MediaObject
	EmbedHTML string
	EmbedURL  string
}

// VideoBlock is a parsed NPF video content block.
type VideoBlock struct {
	URL         string
	Provider    string
	Media       []MediaObject
	EmbedHTML   string
	EmbedIframe *EmbedIframe
	EmbedURL    string
	Poster      []MediaObject
}

// PollResult is a single poll answer's result.
type PollResult struct {
	IsWinner  bool
	VoteCount int
}

// PollResults is the result set returned by the poll callback.
type PollResults struct {
	Timestamp int64
	Results   map[string]PollResult
}

// Answer is a poll answer, kept in source order.
type Answer struct {
	ID   string
	Text string
}

// PollBlock is a parsed NPF poll content block.
type PollBlock struct {
	PollID            string
	Question          string
	Answers           []Answer
	CreationTimestamp int64
	ExpiresAfter      int64
	Votes             *PollResults
	TotalVotes        int
}

// Unsupported is a placeholder for an NPF block type we cannot render.
type Unsupported struct {
	Type string
}

// RowLayout is one row of a "rows" layout.
type RowLayout struct {
	Ranges []int
}

// Rows is a "rows" layout.
type Rows struct {
	Rows          []RowLayout
	TruncateAfter *int
}

// AskLayout is an "ask" layout.
type AskLayout struct {
	Ranges      []int
	Attribution *BlogAttribution
}
