package tumblr

import (
	"encoding/json"
	"os"
	"testing"
)

func loadFixture(t *testing.T, name string) map[string]any {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("parse %s: %v", name, err)
	}
	return out
}

// Fixtures are real (trimmed) Tumblr API responses captured from the public
// API, so these tests exercise the parser against actual data shapes.
func TestParseBlogTimeline(t *testing.T) {
	response := loadFixture(t, "blog_posts.json")

	timeline := ParseBlogTimeline(response, false)
	if timeline == nil {
		t.Fatal("ParseBlogTimeline returned nil")
	}
	if timeline.BlogInfo == nil || timeline.BlogInfo.Name != "staff" {
		t.Fatalf("unexpected blog info: %+v", timeline.BlogInfo)
	}
	if len(timeline.Posts) != 2 {
		t.Fatalf("expected 2 posts, got %d", len(timeline.Posts))
	}

	post := timeline.Posts[0]
	if post.ID == "" || post.PostURL == "" {
		t.Errorf("post missing id/url: %+v", post)
	}
	if len(post.Content) == 0 {
		t.Errorf("post content not parsed")
	}
	if post.Blog == nil || post.Blog.Name == "" {
		t.Errorf("post blog not parsed")
	}
	if post.Date == nil {
		t.Errorf("post date not parsed")
	}
}

func TestParseTimeline(t *testing.T) {
	response := loadFixture(t, "timeline.json")

	timeline := ParseTimeline(response)
	if timeline == nil {
		t.Fatal("ParseTimeline returned nil")
	}
	if len(timeline.Elements) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(timeline.Elements))
	}
	for i, element := range timeline.Elements {
		post, ok := element.(*Post)
		if !ok {
			t.Fatalf("element %d is not a *Post: %T", i, element)
		}
		if post.ID == "" {
			t.Errorf("element %d has no id", i)
		}
	}
}

func TestParseNoteTimelineFromNotes(t *testing.T) {
	// A synthetic "notes" sequence exercises the like/reblog note path.
	response := map[string]any{
		"notes": []any{
			map[string]any{"type": "like", "blogName": "alice", "blogUuid": "t:1", "blogTitle": "Alice", "timestamp": float64(1700000000), "avatarUrl": map[string]any{"512": "https://example.com/a.jpg"}},
			map[string]any{"type": "reblog", "postId": "42", "blogName": "bob", "blogTitle": "Bob", "blogUuid": "t:2", "avatarShape": "circle", "avatarUrl": map[string]any{"512": "https://example.com/b.jpg"}, "tags": []any{"x"}, "reblogParentBlogName": "alice", "timestamp": float64(1700000100)},
		},
		"totalNotes":   float64(2),
		"totalLikes":   float64(1),
		"totalReblogs": float64(1),
		"totalReplies": float64(0),
	}

	timeline := ParseNoteTimeline(response)
	if timeline == nil {
		t.Fatal("ParseNoteTimeline returned nil")
	}
	if len(timeline.Notes) != 2 {
		t.Fatalf("expected 2 notes, got %d", len(timeline.Notes))
	}
	if _, ok := timeline.Notes[0].(*LikeNote); !ok {
		t.Errorf("first note is not a *LikeNote: %T", timeline.Notes[0])
	}
	reblog, ok := timeline.Notes[1].(*ReblogNote)
	if !ok {
		t.Fatalf("second note is not a *ReblogNote: %T", timeline.Notes[1])
	}
	if reblog.Blog == nil || reblog.Blog.Name != "bob" {
		t.Errorf("reblog blog not parsed: %+v", reblog.Blog)
	}
	if timeline.TotalLikes != 1 || timeline.TotalReblogs != 1 {
		t.Errorf("note totals wrong: %+v", timeline)
	}
}
