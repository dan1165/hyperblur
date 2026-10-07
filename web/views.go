package web

import (
	"html"
	"net/url"
	"strings"
	"time"

	"github.com/dan1165/openblur/render"
	"github.com/dan1165/openblur/tumblr"
)

// ---------------------------------------------------------------------------
// Formatting helpers (babel-ish, en_US only)
// ---------------------------------------------------------------------------

// FormatDate renders a date the way babel's "medium" format does.
func FormatDate(t time.Time) string {
	return t.Format("Jan 2, 2006")
}

// FormatDatetime renders a datetime the way babel's "medium" format does.
func FormatDatetime(t time.Time) string {
	return t.Format("Jan 2, 2006, 3:04:05 PM")
}

// formatDecimal groups thousands, matching babel's en_US format_decimal.
func formatDecimal(n int) string {
	return render.Localizer{}.FormatDecimal("", float64(n))
}

// ---------------------------------------------------------------------------
// view builds HTML with explicit escaping of dynamic values.
// ---------------------------------------------------------------------------

type view struct {
	b strings.Builder
}

func (v *view) raw(s string) *view { v.b.WriteString(s); return v }
func (v *view) esc(s string) *view { v.b.WriteString(html.EscapeString(s)); return v }

func htmlEscape(s string) string { return html.EscapeString(s) }

func (v *view) string() string { return v.b.String() }

// PageData is the per-page rendering context.
type PageData struct {
	Path        string
	Endpoint    string
	ExpandPosts bool

	Title       string
	ExtraCSS    []string
	InlineStyle string

	Query string
	Tag   string

	Timeline *tumblr.Timeline
	Blog     *tumblr.BlogTimeline

	Page            int
	PageNumbers     []int // 0 renders an ellipsis
	PostURL         string
	BlogTagName     string
	RequestPollData bool
	QueryArgs       url.Values
	SortBy          string
	PostFilter      string
	TimeFilter      string

	// Blog search page needs the query in the search bar.
	BlogSearchQuery string

	// Notes viewer.
	Notes        *tumblr.NoteTimeline
	NoteType     string
	Latest       bool
	ReblogFilter string
}

// renderPage wraps the given centre content in the shared chrome.
func (a *App) renderPage(data *PageData, center func(v *view)) string {
	v := &view{}
	v.raw("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
	v.raw("<meta charset=\"utf-8\">\n")
	v.raw("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	v.raw("<meta name=\"referrer\" content=\"never\">\n")
	v.raw("<meta name=\"referrer\" content=\"no-referrer\">\n")
	v.raw("<link rel=\"preload\" href=\"/assets/css/base.css\" as=\"style\">\n")
	v.raw("<link rel=\"preload\" href=\"/assets/css/base-post-layout.css\" as=\"style\">\n")
	v.raw("<link rel=\"stylesheet\" type=\"text/css\" href=\"/assets/css/base.css\">\n")
	v.raw("<link rel=\"stylesheet\" type=\"text/css\" href=\"/assets/css/base-post-layout.css\">\n")
	v.raw("<link rel=\"stylesheet\" type=\"text/css\" href=\"/assets/css/post.css\">\n")
	v.raw("<link rel=\"stylesheet\" type=\"text/css\" href=\"/assets/css/post-layout.css\">\n")
	v.raw("<script src=\"/assets/js/base.js\" defer></script>\n")
	v.raw("<script src=\"/assets/js/post.js\" defer></script>\n")
	v.raw("<noscript><style>.with-js {display: none;}</style></noscript>\n")
	v.raw("<script src=\"/assets/js/interaction.js\" defer></script>\n")
	for _, css := range data.ExtraCSS {
		v.raw("<link rel=\"stylesheet\" type=\"text/css\" href=\"")
		v.esc(css)
		v.raw("\">\n")
	}
	if data.InlineStyle != "" {
		v.raw("<style>")
		v.raw(data.InlineStyle)
		v.raw("</style>\n")
	}
	v.raw("<title>")
	v.esc(data.Title)
	v.raw(" ")
	v.esc(a.translate("page_title_suffix"))
	v.raw(" </title>\n</head>\n")
	v.raw("<body class=\"dark-theme\">\n")
	v.raw("<div class=\"container\">\n")
	a.renderNavbar(v, data)
	v.raw("<div class=\"contents\">\n<div class=\"left-column\"></div>\n<div class=\"center-column\">\n")
	center(v)
	v.raw("</div>\n<div class=\"right-column\"></div>\n</div>\n")
	v.raw("<div class=\"buffer\"></div>\n")
	v.raw("<footer class=\"site-footer\"><a href=\"https://github.com/dan1165/openblur\">openblur on GitHub</a></footer>\n")
	v.raw("</div>\n</body>\n</html>\n")
	return v.string()
}

func (a *App) renderNavbar(v *view, data *PageData) {
	searchIcon := `<svg class="icon" xmlns="http://www.w3.org/2000/svg" height="20" viewBox="0 -960 960 960" width="20"><path d="M765-144 526-383q-30 22-65.792 34.5T384.035-336Q284-336 214-406t-70-170q0-100 70-170t170-70q100 0 170 70t70 170.035q0 40.381-12.5 76.173T577-434l239 239-51 51ZM384-408q70 0 119-49t49-119q0-70-49-119t-119-49q-70 0-119 49t-49 119q0 70 49 119t119 49Z"/></svg>`
	v.raw(`<nav class="navbar"><div class="left-section"><a class="logo" href="/explore/trending">openblur</a>`)
	v.raw(`<form class="search-bar" method="get" action="/search" autocomplete="off">`)
	v.raw(searchIcon)

	var value string
	switch {
	case data.Query != "":
		value = data.Query
	case data.Tag != "":
		value = "#" + data.Tag
	}
	v.raw(`<input name="q" type="text"`)
	if value != "" {
		v.raw(` value="`)
		v.esc(value)
		v.raw(`"`)
	}
	v.raw(` placeholder="`)
	v.esc(a.translate("search_bar_placeholder_text"))
	v.raw(`"></input></form></div><div class="center-section">`)
	v.raw(`<a class="nav-tab`)
	if data.Endpoint == "today" {
		v.raw(` selected-tab`)
	}
	v.raw(`" href="/explore/today" title="`)
	v.esc(a.translate("navbar_today_on_tumblr_icon_title"))
	v.raw(`"><svg xmlns="http://www.w3.org/2000/svg" height="30" viewBox="0 -960 960 960" width="30"><path d="M686.588-120q-47.254 0-80.254-33.055-33-33.056-33-80.278 0-46.667 33.078-80t80.333-33.333q47.255 0 80.255 33.333Q800-280 800-233t-33.078 80q-33.079 33-80.334 33Zm-23.254-273.333v-73.333H710v73.333h-46.666Zm0 393.333v-73.333H710V0h-46.666Zm152.333-329.667-33-33.333L835-415.333l33.333 33-52.666 52.666Zm-278 278L505-84.333 557.334-136 590-104l-52.333 52.333Zm309-158.333v-46.666H920V-210h-73.333Zm-393.333 0v-46.666h73.333V-210h-73.333ZM835.667-51.667l-52-53 32.666-32.666 52.334 52-33 33.666ZM556.334-330 505-381.666 537.667-415l51.666 52-32.999 33ZM186.666-80q-27 0-46.833-19.833T120-146.666v-600.001q0-27 19.833-46.833 19.833-19.834 46.833-19.834h56.667V-880h70v66.666h333.334V-880h70v66.666h56.667q27 0 46.833 19.834Q840-773.667 840-746.667v180H186.666v420.001h166.667V-80H186.666Zm0-553.333h586.668v-113.334H186.666v113.334Zm0 0v-113.334 113.334Z"></path></svg></a>`)
	v.raw(`<a class="nav-tab`)
	if data.Endpoint == "trending" {
		v.raw(` selected-tab`)
	}
	v.raw(`" href="/explore/trending" title="`)
	v.esc(a.translate("navbar_trending_icon_title"))
	v.raw(`"><svg xmlns="http://www.w3.org/2000/svg" height="30" viewBox="0 -960 960 960" width="30"><path d="m136-240-56-56 296-298 160 160 208-206H640v-80h240v240h-80v-104L536-320 376-480 136-240Z"/></svg></a>`)
	v.raw(`</div><div class="right-section"></div></nav>`)
}
