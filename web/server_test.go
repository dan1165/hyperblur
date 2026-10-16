package web

import (
	"strings"
	"testing"
)

func TestMinifyCSS(t *testing.T) {
	in := []byte("/* header */\n.a {\n  color: red;\n}\n/* two\n   lines */\n.b { color: blue; }\n")
	want := "\n.a {\ncolor: red;\n}\n\n.b { color: blue; }\n"
	if got := minifyCSS(in); got != want {
		t.Fatalf("minifyCSS() = %q, want %q", got, want)
	}
}

// TestMinifyCSSKeepsRules guards against the minifier eating declarations: the
// inlined sheet must still carry the styles the pages rely on.
func TestMinifyCSSKeepsRules(t *testing.T) {
	assets := loadAssets()
	css := inlineCSS(assets, "css/blog.css")

	if strings.Contains(css, "/*") {
		t.Fatal("comments survived minification")
	}
	if !strings.Contains(css, "\n") {
		t.Fatal("minifier collapsed the whole sheet onto one line")
	}
	for _, want := range []string{
		"content-visibility: auto",                  // feed skips offscreen work
		"--ink-8: hsl(0 0% 7%)",                     // tokens survive
		".post-tag:hover",                           // selectors survive
		"@media (prefers-reduced-motion: reduce) {", // queries are left intact
		"#blog-header",                              // the extra sheet is appended
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("inlined CSS missing %q", want)
		}
	}
	if small := len(css); small > 30000 {
		t.Fatalf("minified CSS is %d bytes, want it well under the raw sheets", small)
	}
}

// TestInlineCSSDropsRetiredTokens catches leftovers from the old palette.
func TestInlineCSSDropsRetiredTokens(t *testing.T) {
	css := inlineCSS(loadAssets(), "")
	for _, gone := range []string{"--color-dt-gray", "--color-top-level-card-bg", "--color-primary-700"} {
		if strings.Contains(css, gone) {
			t.Fatalf("retired token %q still referenced", gone)
		}
	}
}

func TestPageCSSDiffersForBlogPages(t *testing.T) {
	a := &App{assets: loadAssets()}
	a.css = inlineCSS(a.assets, "")
	a.cssBlog = inlineCSS(a.assets, "css/blog.css")

	if a.pageCSS(false) == a.pageCSS(true) {
		t.Fatal("blog pages should get the extra stylesheet")
	}
	if !strings.Contains(a.pageCSS(true), "#blog-header") {
		t.Fatal("blog stylesheet not inlined for blog pages")
	}
	if strings.Contains(a.pageCSS(false), "#blog-header") {
		t.Fatal("blog stylesheet leaked into every page")
	}
}
