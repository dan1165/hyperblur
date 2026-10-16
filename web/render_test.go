package web

import (
	"net/url"
	"strings"
	"testing"
)

func TestURLHandler(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"media stays on the CDN", "https://64.media.tumblr.com/abc/def.jpg", "https://64.media.tumblr.com/abc/def.jpg"},
		{"media keeps query", "https://64.media.tumblr.com/abc/def.jpg?sig=abc123", "https://64.media.tumblr.com/abc/def.jpg?sig=abc123"},
		{"assets stays absolute", "https://assets.tumblr.com/images/x.gif", "https://assets.tumblr.com/images/x.gif"},
		{"static stays absolute", "https://static.tumblr.com/x/y.css", "https://static.tumblr.com/x/y.css"},
		{"audio stays absolute", "https://a.tumblr.com/tumblr_abc.mp3", "https://a.tumblr.com/tumblr_abc.mp3"},
		{"blog post", "https://staff.tumblr.com/post/123/slug", "/staff/123/slug"},
		{"blog page", "https://staff.tumblr.com/tagged/go", "/staff/tagged/go"},
		{"foreign", "https://example.com/x?a=1", "https://example.com/x?a=1"},
		{"t.umblr redirect", "https://t.umblr.com/redirect?z=https%3A%2F%2Fstaff.tumblr.com%2Fpost%2F1%2Fs", "/staff/1/s"},
		{"href.li redirect", "https://href.li/?https://staff.tumblr.com/post/1/s", "/staff/1/s"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := urlHandler(tt.in); got != tt.want {
				t.Fatalf("urlHandler(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestURLHandlerNestedRedirectTerminates guards the recursion cap: wrapped
// redirect links must not recurse without bound.
func TestURLHandlerNestedRedirectTerminates(t *testing.T) {
	target := "https://staff.tumblr.com/post/1/s"
	for i := 0; i < 50; i++ {
		target = "https://href.li/?" + target
	}
	if got := urlHandler(target); got == "" {
		t.Fatal("URLHandler returned empty for deeply nested redirect")
	}
}

func TestDownloadURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		path string
	}{
		{"tumblr media", "https://64.media.tumblr.com/abc/def.jpg", "/download/64/abc/def.jpg"},
		{"foreign", "https://example.com/a.jpg", "/a.jpg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := downloadURL(tt.in)
			parsed, err := url.Parse(got)
			if err != nil {
				t.Fatalf("downloadURL(%q) = %q, unparsable: %v", tt.in, got, err)
			}
			if parsed.Path != tt.path {
				t.Fatalf("downloadURL(%q) path = %q, want %q", tt.in, parsed.Path, tt.path)
			}
		})
	}
}

// TestDownloadURLKeepsExistingQuery is the regression for the doubled "?" bug.
func TestDownloadURLKeepsExistingQuery(t *testing.T) {
	got := downloadURL("https://64.media.tumblr.com/abc/def.jpg?x=1")
	if strings.Count(got, "?") != 1 {
		t.Fatalf("downloadURL produced %q, want a single query separator", got)
	}
	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Path != "/download/64/abc/def.jpg" || parsed.Query().Get("x") != "1" {
		t.Fatalf("downloadURL(%q) = %q, want the download route keeping x=1", "with query", got)
	}
}

func TestPercentEncode(t *testing.T) {
	tests := []struct {
		in, safe, want string
	}{
		{"abc-._~", "", "abc-._~"},
		{"a b/c", "/", "a%20b/c"},
		{"a b/c", "", "a%20b%2Fc"},
		{`x"y`, `"`, `x"y`},
		{"é", "", "%C3%A9"},
	}
	for _, tt := range tests {
		if got := percentEncode(tt.in, tt.safe); got != tt.want {
			t.Fatalf("percentEncode(%q, %q) = %q, want %q", tt.in, tt.safe, got, tt.want)
		}
	}
}

func TestBuildContentDisposition(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"abc/def.jpg", `attachment; filename="def.jpg"; filename*=UTF-8''def.jpg`},
		{`abc/we"ird\na.jpg`, `attachment; filename="weirdna.jpg"; filename*=UTF-8''weirdna.jpg`},
		{"abc/", `attachment; filename="download"; filename*=UTF-8''download`},
	}
	for _, tt := range tests {
		if got := buildContentDisposition(tt.path); got != tt.want {
			t.Fatalf("buildContentDisposition(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}
