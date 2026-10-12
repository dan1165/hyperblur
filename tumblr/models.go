// Package tumblr wraps Tumblr's API, parses JSON into hyperblur models.
package tumblr

import "time"

// Avatar is one blog avatar entry.
type Avatar struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// Blog is Tumblr blog.
type Blog struct {
	Name                  string
	Avatar                []Avatar
	Title                 string
	DescriptionNPF        []any
	Banner                string // focused header image
	Active                bool
	RequiresAccountToView bool
}

// AvatarURL returns highest-res avatar URL, if any.
func (b *Blog) AvatarURL() string {
	if b == nil || len(b.Avatar) == 0 {
		return ""
	}
	return b.Avatar[len(b.Avatar)-1].URL
}

// BrokenBlog is blog Tumblr no longer returns details for.
type BrokenBlog struct {
	Name string
}

// Signpost is advisory card inside timeline.
type Signpost struct {
	Title       string
	Description string
}

// ReblogAttribution describes reblog source.
type ReblogAttribution struct {
	PostID   string
	PostURL  string
	BlogName string
}

// PostTrail is one earlier version of reblogged post.
type PostTrail struct {
	ID         string
	Blog       *Blog
	BrokenBlog *BrokenBlog
	Date       *time.Time
	Content    []any
	Layout     []any
}

// BlogName returns trail blog name, broken or not.
func (t PostTrail) BlogName() string {
	if t.Blog != nil {
		return t.Blog.Name
	}
	if t.BrokenBlog != nil {
		return t.BrokenBlog.Name
	}
	return ""
}

// Post is Tumblr post.
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
}

// ReplyNote is reply on post.
type ReplyNote struct {
	ReplyID string
	Date    *time.Time
	Content []any
	Layout  []any
	Blog    *Blog
}

// ReblogNote is reblog of post.
type ReblogNote struct {
	ID   string
	Blog *Blog

	Content []any
	Layout  []any
	Tags    []string

	Date *time.Time
}

// LikeNote is like on post.
type LikeNote struct {
	BlogName  string
	BlogTitle string
	Date      *time.Time
	// Avatar keyed by size ("128", "512", ...) as Tumblr returns it.
	Avatar map[string]string
}

// BlogTimeline is blog page. Next is continuation cursor, empty on last page.
type BlogTimeline struct {
	BlogInfo   *Blog
	Posts      []*Post
	TotalPosts *int
	Next       string
}

// NoteTimeline is page of post notes.
type NoteTimeline struct {
	Notes []any

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
