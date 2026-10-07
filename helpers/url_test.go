package helpers

import "testing"

// Expected values were captured from openblur's Python url_handler.
func TestURLHandler(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"https://64.media.tumblr.com/abc/s540x810/1.jpg", "/tblr/media/64/abc/s540x810/1.jpg"},
		{"https://49.media.tumblr.com/x.jpg", "/tblr/media/49/x.jpg"},
		{"https://assets.tumblr.com/x.css", "/tblr/assets/x.css"},
		{"https://static.tumblr.com/x.js", "/tblr/static/x.js"},
		{"https://a.tumblr.com/x.mp3", "/tblr/a/x.mp3"},
		{"https://staff.tumblr.com/post/123/slug", "/staff/123/slug"},
		{"https://example.com/x", "https://example.com/x"},
		{"https://href.li/?https://64.media.tumblr.com/a.jpg", "/tblr/media/64/a.jpg"},
		{"https://t.umblr.com/redirect?z=https%3A%2F%2F64.media.tumblr.com%2Fa.jpg&t=x", "/tblr/media/64/a.jpg"},
		{"https://www.tumblr.com/x", "/x"},
	}
	for _, c := range cases {
		if got := URLHandler(c.in); got != c.want {
			t.Errorf("URLHandler(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
