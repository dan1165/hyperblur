package npf

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

type testLocalizer struct{}

func (testLocalizer) Translate(key string, subst map[string]string) string { return key }
func (testLocalizer) TranslatePlural(key string, number int, subst map[string]string) string {
	return key
}
func (testLocalizer) FormatDuration(key string, d time.Duration) string { return "" }
func (testLocalizer) FormatDatetime(key string, t time.Time) string     { return "" }
func (testLocalizer) FormatDecimal(key string, n float64) string {
	return strconv.FormatFloat(n, 'f', -1, 64)
}

func identityURL(u string) string { return u }

func TestValidHexColor(t *testing.T) {
	valid := []string{"#fff", "#ffff", "#ffffff", "#FFFFFFFF", "#0aF3"}
	for _, c := range valid {
		if !validHexColor(c) {
			t.Fatalf("validHexColor(%q) = false, want true", c)
		}
	}
	invalid := []string{"", "fff", "#ff", "#fffff", "red", "#gggggg", "red; background: url(//evil)", "#fff;"}
	for _, c := range invalid {
		if validHexColor(c) {
			t.Fatalf("validHexColor(%q) = true, want false", c)
		}
	}
}

// TestColorFormatDoesNotInjectCSS guarantees inline colour never reaches the
// style attribute unless it is a plain hex colour.
func TestColorFormatDoesNotInjectCSS(t *testing.T) {
	formatter := newInlineFormatter("x", []StyleInterval{
		{Start: 0, End: 1, Instructions: []Instruction{{Type: FMTColor, Hex: `red;background:url(//evil.example)`}}},
	}, identityURL)

	html := formatter.format().Render()
	if strings.Contains(html, "style=") {
		t.Fatalf("non-hex colour reached the style attribute: %q", html)
	}

	formatter = newInlineFormatter("x", []StyleInterval{
		{Start: 0, End: 1, Instructions: []Instruction{{Type: FMTColor, Hex: "#abc123"}}},
	}, identityURL)
	if html := formatter.format().Render(); !strings.Contains(html, "color: #abc123;") {
		t.Fatalf("valid colour dropped: %q", html)
	}
}

func TestCreateSrcsetSkipsZeroWidth(t *testing.T) {
	got := createSrcset([]MediaObject{
		{URL: "a.jpg", Width: 540},
		{URL: "b.jpg", Width: 0},
		{URL: "c.jpg", Width: 1080},
	}, identityURL)

	want := []string{"a.jpg 540w", "c.jpg 1080w"}
	if len(got) != len(want) {
		t.Fatalf("createSrcset = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("createSrcset[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestFormatImageContainerAspectRatio(t *testing.T) {
	block := &ImageBlock{Media: []MediaObject{{URL: "a.jpg", Width: 1080, Height: 540, HasOriginalDimensions: true}}}
	html := formatImageContainer(block, 1, nil, identityURL, testLocalizer{}, false).Render()
	if !strings.Contains(html, "aspect-ratio: 2.0;") {
		t.Fatalf("missing aspect ratio in %q", html)
	}
	if !strings.Contains(html, `srcset="a.jpg 1080w"`) {
		t.Fatalf("missing srcset in %q", html)
	}
}

// TestFormatImageContainerZeroDimensions guards against Inf/NaN aspect ratios
// from media objects the API reports without dimensions.
func TestFormatImageContainerZeroDimensions(t *testing.T) {
	for _, media := range []MediaObject{
		{URL: "a.jpg", Width: 0, Height: 0},
		{URL: "a.jpg", Width: 100, Height: 0},
		{URL: "a.jpg", Width: 0, Height: 100},
	} {
		html := formatImageContainer(&ImageBlock{Media: []MediaObject{media}}, 1, nil, identityURL, testLocalizer{}, false).Render()
		if strings.Contains(html, "Inf") || strings.Contains(html, "NaN") {
			t.Fatalf("invalid aspect ratio for %+v: %q", media, html)
		}
	}
}

func TestApplyLayoutSkipsRatiosForSizelessMedia(t *testing.T) {
	content := []any{
		map[string]any{"type": "image", "media": []any{map[string]any{"url": "a.jpg", "width": float64(0), "height": float64(0)}}},
		map[string]any{"type": "image", "media": []any{map[string]any{"url": "b.jpg", "width": float64(100), "height": float64(0)}}},
	}
	layout := []any{
		map[string]any{"type": "rows", "display": []any{
			map[string]any{"blocks": []any{float64(0), float64(1)}},
		}},
	}

	hasError, html := Format(content, layout, Options{URLHandler: identityURL, Localizer: testLocalizer{}})
	if hasError {
		t.Fatal("render reported an error")
	}
	if strings.Contains(html, "Inf") || strings.Contains(html, "NaN") {
		t.Fatalf("invalid aspect ratio in layout render: %q", html)
	}
}

func TestNodeRenderEscapesAndOrdersAttributes(t *testing.T) {
	node := El("img", "src", `x" onerror=alert(1)`, "alt", "a&b")
	html := node.Render()

	// The quote must be escaped and attributes sorted, so a post cannot break
	// out of the attribute it supplies.
	want := `<img alt="a&amp;b" src="x&quot; onerror=alert(1)">`
	if html != want {
		t.Fatalf("render = %q, want %q", html, want)
	}
}

func TestNodeRenderManyAttributes(t *testing.T) {
	// More attributes than the stack buffer must still render in order.
	node := El("a", "k8", "v8", "k1", "v1", "k5", "v5", "k0", "v0", "k9", "v9", "k2", "v2", "k7", "v7", "k3", "v3", "k6", "v6", "k4", "v4")
	html := node.Render()
	want := `<a k0="v0" k1="v1" k2="v2" k3="v3" k4="v4" k5="v5" k6="v6" k7="v7" k8="v8" k9="v9"></a>`
	if html != want {
		t.Fatalf("render = %q, want %q", html, want)
	}
}

func TestMaxFloatHandlesEmpty(t *testing.T) {
	if got := maxFloat(nil); got != 0 {
		t.Fatalf("maxFloat(nil) = %v, want 0", got)
	}
}
