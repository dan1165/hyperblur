// Package tumblr wraps Tumblr's API and parses its JSON into openblur's models.
package tumblr

import "time"

// Avatar is one entry of a blog's avatar list.
type Avatar struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// Blog is a Tumblr blog.
type Blog struct {
	Name                  string
	Avatar                []Avatar
	Title                 string
	DescriptionNPF        []any
	Banner                string // focused header image
	Active                bool
	RequiresAccountToView bool
}

// AvatarURL returns the highest-resolution avatar URL, if any.
func (b *Blog) AvatarURL() string {
	if b == nil || len(b.Avatar) == 0 {
		return ""
	}
	return b.Avatar[len(b.Avatar)-1].URL
}

// BrokenBlog is a blog that Tumblr no longer returns details for.
type BrokenBlog struct {
	Name string
}

// Signpost is an advisory card shown inside a timeline.
type Signpost struct {
	Title       string
	Description string
}

// CommunityLabel marks potentially sensitive post content.
type CommunityLabel int

const (
	LabelMature CommunityLabel = iota
	LabelDrugUse
	LabelViolence
	LabelSexualThemes
)

// ReblogAttribution describes who a post was reblogged from.
type ReblogAttribution struct {
	PostID    string
	PostURL   string
	BlogName  string
	BlogTitle string
}

// PostTrail is one earlier version of a reblogged post.
type PostTrail struct {
	ID         string
	Blog       *Blog
	BrokenBlog *BrokenBlog
	Date       *time.Time
	Content    []any
	Layout     []any
}

// BlogName returns the trail blog's name regardless of whether it is broken.
func (t PostTrail) BlogName() string {
	if t.Blog != nil {
		return t.Blog.Name
	}
	if t.BrokenBlog != nil {
		return t.BrokenBlog.Name
	}
	return ""
}

// Post is a Tumblr post.
type Post struct {
	Blog *Blog

	ID   string
	Slug string
	Date *time.Time
	Tags []string

	IsAdvertisement bool

	Content []any
	Layout  []any
	Trail   []PostTrail

	NoteCount *int

	DefaultNoteViewerTab string

	ReblogFrom *ReblogAttribution
	ReblogRoot *ReblogAttribution

	CommunityLabels []CommunityLabel
}

// ReplyNote is a reply left on a post.
type ReplyNote struct {
	UUID    string
	ReplyID string
	Date    *time.Time
	Content []any
	Layout  []any
	Blog    *Blog
}

// ReblogNote is a reblog of a post.
type ReblogNote struct {
	UUID string
	ID   string
	Blog *Blog

	Content []any
	Layout  []any
	Tags    []string

	RebloggedFrom string
	Date          *time.Time

	CommunityLabels []CommunityLabel
}

// LikeNote is a like on a post.
type LikeNote struct {
	BlogName  string
	BlogUUID  string
	BlogTitle string
	Date      *time.Time
	// Avatar is keyed by size ("128", "512", ...) as Tumblr returns it.
	Avatar map[string]string
}

// BlogTimeline is a blog page. Next is a continuation cursor, empty when there
// is no next page.
type BlogTimeline struct {
	BlogInfo   *Blog
	Posts      []*Post
	TotalPosts *int
	Next       string
}

// NoteTimeline is a page of post notes.
type NoteTimeline struct {
	Notes []any

	TotalNotes   int
	TotalReplies int
	TotalReblogs int
	TotalLikes   int

	BeforeTimestamp string
	AfterID         string
}

// Timeline is a page of posts and blogs (search, explore, tags).
type Timeline struct {
	Elements  []any
	Signposts []Signpost
	Next      string
}
