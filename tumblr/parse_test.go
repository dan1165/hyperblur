package tumblr

import (
	"net/http"
	"testing"
)

// TestParseReblogNoteKeepsLayout is the regression for reblog notes reading
// their layout out of "content".
func TestParseReblogNoteKeepsLayout(t *testing.T) {
	element := map[string]any{
		"type":    "reblog",
		"postId":  "123",
		"blog":    map[string]any{"name": "someone"},
		"content": []any{map[string]any{"type": "text", "text": "hi"}},
		"layout":  []any{map[string]any{"type": "rows", "display": []any{}}},
	}

	parsed := parseReblogNoteItem(element)
	note, ok := parsed.(*ReblogNote)
	if !ok {
		t.Fatalf("parseReblogNoteItem returned %T, want *ReblogNote", parsed)
	}
	if len(note.Layout) != 1 {
		t.Fatalf("reblog note layout = %v, want the one layout block", note.Layout)
	}
	if len(note.Content) != 1 {
		t.Fatalf("reblog note content = %v, want the one content block", note.Content)
	}
	if note.Blog == nil || note.Blog.Name != "someone" {
		t.Fatalf("reblog note blog = %+v", note.Blog)
	}
}

// TestParseSimpleReblogNoteAvatarOrder pins the largest-last ordering that
// Blog.AvatarURL relies on.
func TestParseSimpleReblogNoteAvatarOrder(t *testing.T) {
	element := map[string]any{
		"type":     "reblog",
		"postId":   "123",
		"blogName": "someone",
		"avatarUrl": map[string]any{
			"512": "https://example.com/512.png",
			"16":  "https://example.com/16.png",
			"128": "https://example.com/128.png",
		},
	}

	parsed := parseReblogNoteItem(element)
	note, ok := parsed.(*ReblogNote)
	if !ok {
		t.Fatalf("parseReblogNoteItem returned %T, want *ReblogNote", parsed)
	}
	if got := note.Blog.AvatarURL(); got != "https://example.com/512.png" {
		t.Fatalf("AvatarURL() = %q, want the 512px avatar", got)
	}
}

// TestAvatarSliceSortsByWidth keeps AvatarURL's "last is largest" contract, and
// the blog header's Avatar[len-2], independent of API ordering.
func TestAvatarSliceSortsByWidth(t *testing.T) {
	raw := []any{
		map[string]any{"url": "big", "width": float64(512), "height": float64(512)},
		map[string]any{"url": "small", "width": float64(16), "height": float64(16)},
		map[string]any{"url": "mid", "width": float64(128), "height": float64(128)},
	}

	blog := parseBlog(map[string]any{"name": "someone", "avatar": raw})
	want := []string{"small", "mid", "big"}
	if len(blog.Avatar) != len(want) {
		t.Fatalf("got %d avatars, want %d", len(blog.Avatar), len(want))
	}
	for i, url := range want {
		if blog.Avatar[i].URL != url {
			t.Fatalf("avatar %d = %q, want %q (ascending width)", i, blog.Avatar[i].URL, url)
		}
	}
	if got := blog.AvatarURL(); got != "big" {
		t.Fatalf("AvatarURL() = %q, want the largest", got)
	}
}

func TestMapErrorResponse(t *testing.T) {
	tests := []struct {
		name       string
		result     map[string]any
		httpStatus int
		wantKind   ErrorKind
	}{
		{
			name:       "not found without errors array",
			result:     map[string]any{"meta": map[string]any{"status": float64(http.StatusNotFound), "msg": "Not Found"}},
			httpStatus: http.StatusNotFound,
			wantKind:   ErrBlogNotFound,
		},
		{
			name:       "server error stays generic",
			result:     map[string]any{"meta": map[string]any{"status": float64(http.StatusInternalServerError), "msg": "Server Error"}},
			httpStatus: http.StatusInternalServerError,
			wantKind:   ErrGeneric,
		},
		{
			name: "login required",
			result: map[string]any{
				"meta":   map[string]any{"status": float64(http.StatusForbidden), "msg": "Forbidden"},
				"errors": []any{map[string]any{"code": float64(4012), "detail": "Login required"}},
			},
			httpStatus: http.StatusForbidden,
			wantKind:   ErrLoginRequired,
		},
		{
			name: "restricted tag",
			result: map[string]any{
				"meta":   map[string]any{"status": float64(http.StatusForbidden), "msg": "Forbidden"},
				"errors": []any{map[string]any{"code": float64(13001), "detail": "Restricted"}},
			},
			httpStatus: http.StatusForbidden,
			wantKind:   ErrRestrictedTag,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mapErrorResponse(tt.result, tt.httpStatus)
			if err.Kind != tt.wantKind {
				t.Fatalf("kind = %v, want %v", err.Kind, tt.wantKind)
			}
			if err.Message == "" {
				t.Fatal("message is empty")
			}
		})
	}
}

func TestCursorFrom(t *testing.T) {
	raw := map[string]any{
		"links": map[string]any{
			"next": map[string]any{"queryParams": map[string]any{"cursor": "abc"}},
		},
	}
	if got := cursorFrom(raw); got != "abc" {
		t.Fatalf("cursorFrom = %q, want abc", got)
	}
	if got := cursorFrom(map[string]any{}); got != "" {
		t.Fatalf("cursorFrom(empty) = %q, want empty", got)
	}
}
