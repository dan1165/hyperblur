package web

import (
	"testing"

	"github.com/dan1165/hyperblur/tumblr"
)

func TestValidMediaPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"", true},
		{"a/b.jpg", true},
		{"a//b.jpg", true},
		{"a/..b/c.jpg", true},
		{"a/b..jpg", true},
		{"..", false},
		{"../a", false},
		{"a/../b", false},
		{"a/..", false},
		{"%2e%2e/x", false},
		{"a/%2E%2E%2Fb", false},
		{"a%00b", false},
	}
	for _, tt := range tests {
		if got := validMediaPath(tt.path); got != tt.want {
			t.Fatalf("validMediaPath(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestValidPollID(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{"1234567890", true},
		{"abc_DEF-123", true},
		{"", false},
		{"..", false},
		{"1/2", false},
		{"1%2F2", false},
		{"a b", false},
		{string(make([]byte, 65)), false},
	}
	for _, tt := range tests {
		if got := validPollID(tt.id); got != tt.want {
			t.Fatalf("validPollID(%q) = %v, want %v", tt.id, got, tt.want)
		}
	}
}

func TestValidBlogName(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"staff", true},
		{"my-blog-2", true},
		{"", true}, // callers reject empty separately
		{"-staff", false},
		{"staff-", false},
		{"Sta ff", false},
		{"../../etc", false},
	}
	for _, tt := range tests {
		if got := validBlogName(tt.name); got != tt.want {
			t.Fatalf("validBlogName(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestNumericPostID(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{"123456", true},
		{"-1", true}, // a number, rejected upstream rather than by us
		{"", false},
		{"abc", false},
		{"1/2", false},
	}
	for _, tt := range tests {
		if got := numericPostID(tt.id); got != tt.want {
			t.Fatalf("numericPostID(%q) = %v, want %v", tt.id, got, tt.want)
		}
	}
}

func TestMediaUpstreamQuery(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"download=1", ""},
		{"download=1&download=1", ""},
		{"foo=1&download=1&bar=2", "foo=1&bar=2"},
		{"foo=1", "foo=1"},
		{"a=%20&b=2", "a=%20&b=2"},
	}
	for _, tt := range tests {
		if got := mediaUpstreamQuery(tt.in); got != tt.want {
			t.Fatalf("mediaUpstreamQuery(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestParsePostFilter(t *testing.T) {
	if got := parsePostFilter("photo"); got != tumblr.FilterPhoto {
		t.Fatalf("parsePostFilter(photo) = %q", got)
	}
	if got := parsePostFilter("ask"); got != tumblr.FilterAnswer {
		t.Fatalf("parsePostFilter(ask) = %q", got)
	}
	if got := parsePostFilter("nonsense"); got != "" {
		t.Fatalf("parsePostFilter(nonsense) = %q, want empty", got)
	}
}

func TestNormalizeTimeFilter(t *testing.T) {
	for _, ok := range []string{"1", "7", "30", "180", "365"} {
		if got := normalizeTimeFilter(ok); got != ok {
			t.Fatalf("normalizeTimeFilter(%q) = %q", ok, got)
		}
	}
	for _, bad := range []string{"", "0", "999", "1; drop"} {
		if got := normalizeTimeFilter(bad); got != "0" {
			t.Fatalf("normalizeTimeFilter(%q) = %q, want 0", bad, got)
		}
	}
}
