package web

import (
	"html"
	"net/url"
	"strings"
	"time"

	"github.com/dan1165/hyperblur/tumblr"
)

// ---------------------------------------------------------------------------
// Formatting helpers (babel-ish, en_US only)
// ---------------------------------------------------------------------------

// formatDate renders a date the way babel's "medium" format does.
func formatDate(t time.Time) string {
	return t.Format("Jan 2, 2006")
}

// formatDatetime renders a datetime the way babel's "medium" format does.
func formatDatetime(t time.Time) string {
	return t.Format("Jan 2, 2006, 3:04:05 PM")
}

// formatDecimal groups thousands, matching babel's en_US format_decimal.
func formatDecimal(n int) string {
	return Localizer{}.FormatDecimal("", float64(n))
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
	Path     string
	Endpoint string

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
	v.b.Grow(16384)
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
	v.raw("<script src=\"/assets/js/post.js\" defer></script>\n")
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
	v.esc("- hyperblur")
	v.raw(" </title>\n</head>\n")
	v.raw("<body class=\"dark-theme\">\n")
	v.raw("<div class=\"container\">\n")
	a.renderNavbar(v, data)
	v.raw("<div class=\"contents\">\n<div class=\"left-column\"></div>\n<div class=\"center-column\">\n")
	center(v)
	v.raw("</div>\n<div class=\"right-column\"></div>\n</div>\n")
	v.raw("<div class=\"buffer\"></div>\n")
	v.raw("<footer class=\"site-footer\"><a href=\"https://github.com/dan1165/hyperblur\">hyperblur on GitHub</a></footer>\n")
	v.raw("</div>\n</body>\n</html>\n")
	return v.string()
}

func (a *App) renderNavbar(v *view, data *PageData) {
	searchIcon := `<svg class="icon" xmlns="http://www.w3.org/2000/svg" height="20" viewBox="0 -960 960 960" width="20"><path d="M765-144 526-383q-30 22-65.792 34.5T384.035-336Q284-336 214-406t-70-170q0-100 70-170t170-70q100 0 170 70t70 170.035q0 40.381-12.5 76.173T577-434l239 239-51 51ZM384-408q70 0 119-49t49-119q0-70-49-119t-119-49q-70 0-119 49t-49 119q0 70 49 119t119 49Z"/></svg>`
	v.raw(`<nav class="navbar"><div class="left-section"><a class="logo" href="/explore/trending">hyperblur</a>`)
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
	v.esc("Search")
	v.raw(`"></input></form></div><div class="center-section">`)
	v.raw(`<a class="nav-tab`)
	if data.Endpoint == "today" {
		v.raw(` selected-tab`)
	}
	v.raw(`" href="/explore/today" title="`)
	v.esc("Today on Tumblr")
	v.raw(`"><svg xmlns="http://www.w3.org/2000/svg" height="30" viewBox="0 -960 960 960" width="30"><path d="M686.588-120q-47.254 0-80.254-33.055-33-33.056-33-80.278 0-46.667 33.078-80t80.333-33.333q47.255 0 80.255 33.333Q800-280 800-233t-33.078 80q-33.079 33-80.334 33Zm-23.254-273.333v-73.333H710v73.333h-46.666Zm0 393.333v-73.333H710V0h-46.666Zm152.333-329.667-33-33.333L835-415.333l33.333 33-52.666 52.666Zm-278 278L505-84.333 557.334-136 590-104l-52.333 52.333Zm309-158.333v-46.666H920V-210h-73.333Zm-393.333 0v-46.666h73.333V-210h-73.333ZM835.667-51.667l-52-53 32.666-32.666 52.334 52-33 33.666ZM556.334-330 505-381.666 537.667-415l51.666 52-32.999 33ZM186.666-80q-27 0-46.833-19.833T120-146.666v-600.001q0-27 19.833-46.833 19.833-19.834 46.833-19.834h56.667V-880h70v66.666h333.334V-880h70v66.666h56.667q27 0 46.833 19.834Q840-773.667 840-746.667v180H186.666v420.001h166.667V-80H186.666Zm0-553.333h586.668v-113.334H186.666v113.334Zm0 0v-113.334 113.334Z"></path></svg></a>`)
	v.raw(`<a class="nav-tab`)
	if data.Endpoint == "trending" {
		v.raw(` selected-tab`)
	}
	v.raw(`" href="/explore/trending" title="`)
	v.esc("Trending")
	v.raw(`"><svg xmlns="http://www.w3.org/2000/svg" height="30" viewBox="0 -960 960 960" width="30"><path d="m136-240-56-56 296-298 160 160 208-206H640v-80h240v240h-80v-104L536-320 376-480 136-240Z"/></svg></a>`)
	v.raw(`</div><div class="right-section"></div></nav>`)
}

// ---------------------------------------------------------------------------
// Posts
// ---------------------------------------------------------------------------

func (a *App) renderPost(v *view, data *PageData, post *tumblr.Post) {
	v.raw(`<div class="post" data-post-id="`)
	v.esc(post.ID)
	v.raw(`">`)

	postURL := data.PostURL
	if postURL == "" {
		postURL = PostPath(post)
	}

	a.renderPostHeader(v, data, postHeaderData{
		Blog:           post.Blog,
		Date:           post.Date,
		ReblogFrom:     post.ReblogFrom,
		ReblogRoot:     post.ReblogRoot,
		UseThisPostURL: postURL,
	})
	a.renderPostBody(v, data, post)
	a.renderPostFooter(v, data, post, postURL)

	v.raw(`</div>`)
}

func (a *App) renderPostBody(v *view, data *PageData, post *tumblr.Post) {
	hasContent := len(post.Content) > 0

	var mainErr *RenderError
	var mainTag string
	if hasContent {
		mainErr, mainTag = a.FormatNPF(post.Content, post.Layout, post.Blog.Name, post.ID, data.RequestPollData)
	}

	v.raw(`<div class="post-content">`)

	if len(post.Trail) > 0 {
		v.raw(`<div class="post-trails">`)
	}

	for _, trail := range post.Trail {
		v.raw(`<div class="trail-post">`)
		a.renderPostHeader(v, data, postHeaderData{
			Blog:           trail.Blog,
			BrokenBlog:     trail.BrokenBlog,
			Date:           trail.Date,
			UseThisPostURL: trail.BlogName() + "/" + trail.ID,
		})
		trailErr, trailTag := a.FormatNPF(trail.Content, trail.Layout, post.Blog.Name, post.ID, data.RequestPollData)
		if trailErr != nil {
			a.logNPFError(post, "trail", trailErr)
			a.renderNPFError(v, trailErr)
		}
		v.raw(trailTag)
		v.raw(`</div>`)
	}

	if len(post.Trail) > 0 && hasContent {
		postURL := data.PostURL
		if postURL == "" {
			postURL = PostPath(post)
		}
		v.raw(`<div class="trail-post">`)
		a.renderPostHeader(v, data, postHeaderData{
			Blog:           post.Blog,
			Date:           post.Date,
			ReblogFrom:     post.ReblogFrom,
			ReblogRoot:     post.ReblogRoot,
			SkipReblog:     true,
			UseThisPostURL: postURL,
		})
		if mainErr != nil {
			a.logNPFError(post, "post", mainErr)
			a.renderNPFError(v, mainErr)
		}
		v.raw(mainTag)
		v.raw(`</div>`)
	} else {
		if mainErr != nil {
			a.logNPFError(post, "post", mainErr)
			a.renderNPFError(v, mainErr)
		}
		v.raw(mainTag)
	}

	if len(post.Trail) > 0 {
		v.raw(`</div>`)
	}
	v.raw(`</div>`)
}

func (a *App) renderPostFooter(v *view, data *PageData, post *tumblr.Post, postURL string) {
	v.raw(`<footer class="post-footer">`)

	renderTags(v, post.Tags, data.BlogTagName)

	noteCount := 0
	if post.NoteCount != nil {
		noteCount = *post.NoteCount
	}

	v.raw(`<div class="post-interaction"><div class="note-count"><a href="/`)
	v.esc(postURL)
	v.raw(`?note_viewer=`)
	v.esc(post.DefaultNoteViewerTab)
	v.raw(`"> `)
	v.esc(noteCountLabel(noteCount))
	v.raw(`</a></div><div class="interaction-buttons">`)
	v.raw(`<a href="https://www.tumblr.com/`)
	v.esc(postURL)
	v.raw(`" rel="noreferrer"><svg xmlns="http://www.w3.org/2000/svg" height="18px" viewBox="0 0 320 512" role="img" aria-label="`)
	v.esc("View on Tumblr")
	v.raw(`"><title>`)
	v.esc("View on Tumblr")
	v.raw(`</title><path d="M309.8 480.3c-13.6 14.5-50 31.7-97.4 31.7-120.8 0-147-88.8-147-140.6v-144H17.9c-5.5 0-10-4.5-10-10v-68c0-7.2 4.5-13.6 11.3-16 62-21.8 81.5-76 84.3-117.1.8-11 6.5-16.3 16.1-16.3h70.9c5.5 0 10 4.5 10 10v115.2h83c5.5 0 10 4.4 10 9.9v81.7c0 5.5-4.5 10-10 10h-83.4V360c0 34.2 23.7 53.6 68 35.8 4.8-1.9 9-3.2 12.7-2.2 3.5.9 5.8 3.4 7.4 7.9l22 64.3c1.8 5 3.3 10.6-.4 14.5z"/></svg></a>`)
	v.raw(`</div></div></footer>`)
}

func (a *App) renderNPFError(v *view, renderErr *RenderError) {
	v.raw(`<figure class="alert error secondary"><figcaption><svg class="icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 -960 960 960"><path d="M480-280q17 0 28.5-11.5T520-320q0-17-11.5-28.5T480-360q-17 0-28.5 11.5T440-320q0 17 11.5 28.5T480-280Zm-40-160h80v-240h-80v240Zm40 360q-83 0-156-31.5T197-197q-54-54-85.5-127T80-480q0-83 31.5-156T197-763q54-54 127-85.5T480-880q83 0 156 31.5T763-763q54 54 85.5 127T880-480q0 83-31.5 156T763-197q-54 54-127 85.5T480-80Z"/></svg><h4>`)
	v.esc("Failed to render the contents of this post")
	v.raw(`</h4><p>`)
	v.esc("The contents of this post has failed to render due to an error. Check below for more information")
	v.raw(`</p></figcaption>`)
	a.renderErrorDetails(v, renderErr.Name, renderErr.Message, "", false)
	v.raw(`</figure>`)
}

// logNPFError records an NPF render failure for a post.
func (a *App) logNPFError(post *tumblr.Post, where string, err *RenderError) {
	a.Logger.Printf("npf render error (%s) for post %s/%s: %s", where, post.Blog.Name, post.ID, err.Message)
}

func (a *App) renderErrorDetails(v *view, name, message, context string, open bool) {
	v.raw(`<details`)
	if open {
		v.raw(` open=""`)
	}
	v.raw(` class="error-technical-details"><summary>`)
	v.esc("Show error log")
	v.raw(`</summary><pre>Error: `)
	v.esc(name)
	v.raw(`</pre>`)
	if message != "" {
		v.raw(`<pre>Message: "`)
		v.esc(message)
		v.raw(`"</pre>`)
	}
	if context != "" {
		v.raw(`<pre>Context:<br/><br/>`)
		v.esc(context)
		v.raw(`</pre>`)
	}
	v.raw(`</details>`)
}

// ---------------------------------------------------------------------------
// Timelines
// ---------------------------------------------------------------------------

func (a *App) renderTimelineCenter(v *view, data *PageData, controlBar func(v *view), paging func(v *view)) {
	v.raw(`<ul id="timeline-control-bar">`)
	if controlBar != nil {
		controlBar(v)
	}
	v.raw(`</ul>`)

	if len(data.Timeline.Signposts) > 0 {
		a.renderAlerts(v, data)
	}

	v.raw(`<div class="timeline" id="m">`)
	for _, element := range data.Timeline.Elements {
		if post, ok := element.(*tumblr.Post); ok && !post.IsAdvertisement {
			a.renderPost(v, data, post)
		}
	}
	v.raw(`</div>`)

	if paging != nil && data.Timeline.Next != "" {
		v.raw(`<div class="paging">`)
		paging(v)
		v.raw(`</div>`)
	}
}

func (a *App) renderAlerts(v *view, data *PageData) {
	v.raw(`<div class="alerts">`)
	for _, signpost := range data.Timeline.Signposts {
		switch signpost.Title {
		case "Hold your horses!":
			v.raw(`<figure class="alert"><figcaption><h4>`)
			v.esc("Restricted search results")
			v.raw(`</h4><p>`)
			v.esc("These search results have been blocked by Tumblr for violating their content guidelines.")
			v.raw(`</p></figcaption></figure>`)
		case "Woah, hang on there.":
			v.raw(`<figure class="alert"><figcaption><h4>`)
			v.esc("Quick Update on Your Search Results")
			v.raw(`</h4><p>`)
			v.esc("These search results have been partially restricted due to Tumblr's content guidelines. You can expect to see some empty results here sporadically.")
			v.raw(`</p></figcaption></figure>`)
		default:
			v.raw(`<figure class="tumblr-signpost"><svg class="icon" xmlns="http://www.w3.org/2000/svg" height="48px" width="48px" viewBox="0 -960 960 960"><path d="M451.5-85v-185H234L125-379l109-108.5h217.5v-90H165V-795h286.5v-80H509v80h217.5L835-686.5l-108.5 109H509v90h286V-270H509v185h-57.5Z"/></svg><div><h3>`)
			v.esc(signpost.Title)
			v.raw(`</h3><p>`)
			v.esc(signpost.Description)
			v.raw(`</p></div></figure>`)
		}
	}
	v.raw(`</div>`)
}

// ---------------------------------------------------------------------------
// Blog header
// ---------------------------------------------------------------------------

func (a *App) renderBlogHeader(v *view, data *PageData) {
	blogInfo := data.Blog.BlogInfo

	var descriptionTag string
	if len(blogInfo.DescriptionNPF) > 0 {
		if _, tag := a.FormatNPF(blogInfo.DescriptionNPF, nil, "", "", false); tag != "" {
			descriptionTag = tag
		}
	}

	banner := URLHandler(blogInfo.Banner)
	avatar := ""
	if len(blogInfo.Avatar) >= 2 {
		avatar = URLHandler(blogInfo.Avatar[len(blogInfo.Avatar)-2].URL)
	} else if len(blogInfo.Avatar) == 1 {
		avatar = URLHandler(blogInfo.Avatar[0].URL)
	}

	v.raw(`<header id="blog-header"><img id="banner" alt="`)
	v.esc("Blog banner")
	v.raw(`" src="`)
	v.esc(banner)
	v.raw(`"/><a href="/`)
	v.esc(blogInfo.Name)
	v.raw(`"><img class="avatar" alt="`)
	v.esc("Blog avatar")
	v.raw(`" src="`)
	v.esc(avatar)
	v.raw(`"/></a><div class="blog-header-textual-content">`)
	if blogInfo.Title != "" {
		v.raw(`<h1 id="blog-title">`)
		v.esc(blogInfo.Title)
		v.raw(`</h1>`)
	}
	v.raw(`<p class="blog-name"><a href="/`)
	v.esc(blogInfo.Name)
	v.raw(`">@`)
	v.esc(blogInfo.Name)
	v.raw(`</a></p><div id="blog-description">`)
	v.raw(descriptionTag)
	v.raw(`</div></div></header>`)
}

// noteCountLabel renders a post's note count, e.g. "1 note" / "5 notes".
func noteCountLabel(n int) string {
	if n == 1 {
		return "1 note"
	}
	return formatDecimal(n) + " notes"
}

// percentEncode percent-encodes s, leaving unreserved characters and any
// characters in safe untouched.
func percentEncode(s, safe string) string {
	const unreserved = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~"
	const hex = "0123456789ABCDEF"
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if strings.IndexByte(unreserved, c) >= 0 || strings.IndexByte(safe, c) >= 0 {
			b.WriteByte(c)
			continue
		}
		b.WriteByte('%')
		b.WriteByte(hex[c>>4])
		b.WriteByte(hex[c&0xf])
	}
	return b.String()
}

// urlEscape percent-encodes a single path segment.
func urlEscape(s string) string {
	return percentEncode(s, "")
}

// Icons reused from the templates.
const (
	reblogIcon   = `<svg class="icon" xmlns="http://www.w3.org/2000/svg" height="16" viewBox="0 -960 960 960" width="16"><path d="M280-80 120-240l160-160 56 58-62 62h406v-160h80v240H274l62 62-56 58Zm-80-440v-240h486l-62-62 56-58 160 160-160 160-56-58 62-62H280v160h-80Z"/></svg>`
	dropdownIcon = `<svg class="icon" xmlns="http://www.w3.org/2000/svg" height="16" viewBox="0 -960 960 960" width="16"><path d="M480-345 240-585l56-56 184 184 184-184 56 56-240 240Z"/></svg>`
	heartIcon    = `<svg class="icon" xmlns="http://www.w3.org/2000/svg" height="24" viewBox="0 -960 960 960" width="24"><path d="M479.62-171.62q-12.85 0-25.81-4.61-12.96-4.62-22.81-14.46l-57.46-52.23q-106.38-97-189.96-190.58Q100-527.08 100-634q0-85.15 57.42-142.58Q214.85-834 300-834q48.38 0 95.58 22.31 47.19 22.31 84.42 72.46 37.23-50.15 84.42-72.46Q611.62-834 660-834q85.15 0 142.58 57.42Q860-719.15 860-634q0 108.08-85 202.73-85 94.65-189.54 188.96l-56.85 51.62q-9.84 9.84-22.99 14.46-13.16 4.61-26 4.61Z"/></svg>`
	replyIcon    = `<svg class="icon" xmlns="http://www.w3.org/2000/svg" height="24" viewBox="0 -960 960 960" width="24"><path d="M80-80v-733.33q0-27 19.83-46.84Q119.67-880 146.67-880h666.66q27 0 46.84 19.83Q880-840.33 880-813.33v506.66q0 27-19.83 46.84Q840.33-240 813.33-240H240L80-80Zm131.33-226.67h602v-506.66H146.67v575l64.66-68.34Zm-64.66 0v-506.66 506.66Z"/></svg>`
)

type postHeaderData struct {
	Blog           *tumblr.Blog
	BrokenBlog     *tumblr.BrokenBlog
	Date           *time.Time
	ReblogFrom     *tumblr.ReblogAttribution
	ReblogRoot     *tumblr.ReblogAttribution
	SkipReblog     bool
	UseThisPostURL string
}

// PostPath returns the post's path (without a leading slash).
func PostPath(post *tumblr.Post) string {
	path := post.Blog.Name + "/" + post.ID
	if post.Slug != "" {
		path += "/" + post.Slug
	}
	return path
}

func (a *App) renderPostHeader(v *view, data *PageData, hd postHeaderData) {
	v.raw(`<div class="post-header"><div class="post-author">`)

	renderAvatar(v, hd.Blog)

	v.raw(`<div class="author-information"><div class="primary-post-author">`)
	renderBlogName(v, hd.Blog, hd.BrokenBlog)
	renderPostDate(v, hd.Date, hd.UseThisPostURL)
	v.raw(`</div>`)

	if hd.ReblogFrom != nil && !hd.SkipReblog {
		v.raw(`<div class="reblog-attribution">`)
		v.raw(reblogIcon)
		v.raw(a.reblogAttribution(hd.ReblogFrom, hd.ReblogRoot))
		v.raw(`</div>`)
	}

	v.raw(`</div></div></div>`)
}

// renderAvatarLink renders a linked blog avatar.
func renderAvatarLink(v *view, href, src string) {
	v.raw(`<a href="/`)
	v.esc(href)
	v.raw(`"><img class="avatar" alt="Blog avatar" loading="lazy" src="`)
	v.esc(src)
	v.raw(`"></a>`)
}

// renderAvatar renders a blog's avatar, or an anonymous one when the blog is
// inactive or missing.
func renderAvatar(v *view, blog *tumblr.Blog) {
	if blog != nil && blog.Active {
		renderAvatarLink(v, blog.Name, URLHandler(blog.AvatarURL()))
		return
	}
	v.raw(`<img class="avatar" alt="Blog avatar" loading="lazy" src="/assets/images/anon_96px.png">`)
}

// renderBlogName renders a blog's name, styled by whether the blog is active,
// deactivated or broken.
func renderBlogName(v *view, blog *tumblr.Blog, broken *tumblr.BrokenBlog) {
	switch {
	case blog != nil && !blog.Active:
		v.raw(`<span class="link blog-name deactivated-blog">`)
		v.esc(blog.Name)
		v.raw(`</span>`)
	case blog != nil:
		v.raw(`<a class="link blog-name" href="/`)
		v.esc(blog.Name)
		v.raw(`">`)
		v.esc(blog.Name)
		v.raw(`</a>`)
	case broken != nil:
		v.raw(`<span class="link blog-name broken-blog" href="/`)
		v.esc(broken.Name)
		v.raw(`">`)
		v.esc(broken.Name)
		v.raw(`</span>`)
	}
}

// renderPostDate renders a post timestamp, linked to postPath when given.
func renderPostDate(v *view, date *time.Time, postPath string) {
	if date == nil {
		return
	}
	v.raw(`<span class="separator">•</span><span class="post-timestamp" title="`)
	v.esc(formatDatetime(*date))
	v.raw(`">`)
	if postPath != "" {
		v.raw(`<a href="/`)
		v.esc(postPath)
		v.raw(`">`)
	}
	v.raw(`<time datetime="`)
	v.esc(date.Format("2006-01-02T15:04"))
	v.raw(`">`)
	v.esc(formatDate(*date))
	v.raw(`</time>`)
	if postPath != "" {
		v.raw(`</a>`)
	}
	v.raw(`</span>`)
}

// renderTags renders a post's tags, scoped to blogName's tag page when given.
func renderTags(v *view, tags []string, blogName string) {
	if len(tags) == 0 {
		return
	}
	v.raw(`<div class="post-tags">`)
	for _, tag := range tags {
		v.raw(`<a class="post-tag" href="/`)
		if blogName != "" {
			v.esc(blogName)
			v.raw(`/tagged/`)
		} else {
			v.raw(`tagged/`)
		}
		v.raw(urlEscape(tag))
		v.raw(`"><span>#`)
		v.esc(tag)
		v.raw(`</span></a>`)
	}
	v.raw(`</div>`)
}

// reblogAttribution renders who the post was reblogged from.
func (a *App) reblogAttribution(from, root *tumblr.ReblogAttribution) string {
	classes := []string{"link", "blog-name"}
	name := from.BlogName
	if name == "" {
		name = "reblogged"
		classes = append(classes, "hidden-reblog")
	}

	url := from.PostURL
	if !isTumblrURL(url) {
		if root != nil && root.PostID == from.PostID && root.BlogName != "" {
			url = "/" + root.BlogName + "/" + from.PostID
		} else {
			return `<span class="` + strings.Join(classes, " ") + `">` + htmlEscape(name) + `</span>`
		}
	}

	return `<a href="` + htmlEscape(URLHandler(url)) + `" class="` + strings.Join(classes, " ") + `">` + htmlEscape(name) + `</a>`
}

func isTumblrURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == "tumblr.com" || strings.HasSuffix(host, ".tumblr.com")
}

// renderNotesViewer writes the post-notes section for a post.
func (a *App) renderNotesViewer(v *view, data *PageData) {
	notes := data.Notes
	if notes == nil {
		return
	}

	v.raw(`<section class="post-notes"><header><ul class="post-notes-nav">`)
	a.notesTab(v, data, "replies", replyIcon, "Replies", notes.TotalReplies)
	a.notesTab(v, data, "reblogs", reblogIcon, "Reblogs", notes.TotalReblogs)
	a.notesTab(v, data, "likes", heartIcon, "Likes", notes.TotalLikes)
	v.raw(`</ul><ul id="timeline-control-bar">`)
	a.renderNotesControlBar(v, data)
	v.raw(`</ul></header><section>`)
	if len(notes.Notes) > 0 {
		for _, note := range notes.Notes {
			switch n := note.(type) {
			case *tumblr.ReplyNote:
				a.renderReplyNote(v, data, n)
			case *tumblr.ReblogNote:
				a.renderReblogNote(v, data, n)
			case *tumblr.LikeNote:
				a.renderLikeNote(v, data, n)
			}
		}
	}
	v.raw(`</section><footer>`)
	a.renderNotesPaging(v, data)
	v.raw(`</footer></section>`)
}

func (a *App) notesTab(v *view, data *PageData, noteType, icon, title string, count int) {
	v.raw(`<li`)
	if data.NoteType == noteType {
		v.raw(` class="selected"`)
	}
	v.raw(`><a href="/`)
	v.esc(data.PostURL)
	v.raw(`?note_viewer=`)
	v.raw(noteType)
	v.raw(`" title="`)
	v.esc(title)
	v.raw(`">`)
	v.raw(icon)
	v.raw(`<span>`)
	v.esc(formatDecimal(count))
	v.raw(`</span></a></li>`)
}

func (a *App) renderNotesControlBar(v *view, data *PageData) {
	switch data.NoteType {
	case "replies":
		v.raw(`<li class="control-bar-action no-js" id="sort-by-filter" title="`)
		v.esc("Sort by")
		v.raw(`"><span>`)
		v.esc("Filter")
		v.raw(dropdownIcon)
		v.raw(`</span><ul class="control-bar-dropdown-menu">`)
		v.raw(searchFilterItem(!data.Latest, "/"+data.PostURL+"?note_viewer=replies", "Oldest first"))
		v.raw(searchFilterItem(data.Latest, "/"+data.PostURL+"?note_viewer=replies&latest", "Newest first"))
		v.raw(`</ul></li>`)
	case "reblogs":
		filter := data.ReblogFilter
		if filter == "" {
			filter = "reblogs_with_comments"
		}
		v.raw(`<li class="control-bar-action no-js" id="sort-by-filter" title="`)
		v.esc("Sort by")
		v.raw(`"><span>`)
		v.esc("Filter")
		v.raw(dropdownIcon)
		v.raw(`</span><ul class="control-bar-dropdown-menu">`)
		v.raw(searchFilterItem(filter == "reblogs_with_comments", "/"+data.PostURL+"?note_viewer=reblogs", "Comments and tags"))
		v.raw(searchFilterItem(filter == "reblogs_with_content_comments", "/"+data.PostURL+"?note_viewer=reblogs&reblog_filter=reblogs_with_content_comments", "Comments only"))
		v.raw(searchFilterItem(filter == "reblogs_only", "/"+data.PostURL+"?note_viewer=reblogs&reblog_filter=reblogs_only", "Other reblogs"))
		v.raw(`</ul></li>`)
	}
}

func (a *App) renderNotesPaging(v *view, data *PageData) {
	notes := data.Notes
	if notes == nil {
		return
	}
	switch data.NoteType {
	case "replies":
		if notes.AfterID != "" {
			v.raw(`<a class="secondary next-page button" href="/`)
			v.esc(data.PostURL)
			v.raw(`?`)
			v.raw(htmlEscape(updateQuery(data.QueryArgs, "after", notes.AfterID)))
			v.raw(`">`)
			v.esc("Next page")
			v.raw(`</a>`)
		}
	default:
		if notes.BeforeTimestamp != "" {
			v.raw(`<a class="secondary next-page button" href="/`)
			v.esc(data.PostURL)
			v.raw(`?`)
			v.raw(htmlEscape(updateQuery(data.QueryArgs, "before_timestamp", notes.BeforeTimestamp)))
			v.raw(`">`)
			v.esc("Next page")
			v.raw(`</a>`)
		}
	}
}

func (a *App) renderReplyNote(v *view, data *PageData, note *tumblr.ReplyNote) {
	blog := note.Blog

	v.raw(`<div class="note reply" data-id="`)
	v.esc(note.ReplyID)
	v.raw(`">`)

	renderAvatar(v, blog)

	v.raw(`<div><div class="author-information"><div class="primary-post-author">`)
	renderBlogName(v, blog, nil)
	renderPostDate(v, note.Date, "")
	v.raw(`</div></div><p>`)

	blogName := ""
	if blog != nil {
		blogName = blog.Name
	}
	_, tag := a.FormatNPF(note.Content, note.Layout, blogName, "", data.RequestPollData)
	v.raw(tag)
	v.raw(`</p></div></div>`)
}

func (a *App) renderReblogNote(v *view, data *PageData, note *tumblr.ReblogNote) {
	// The template only ever renders the "blog name absent" branch, since
	// reblog notes never expose a top-level blog_name.
	v.raw(`<div class="note reblog-note post" data-post-id="`)
	v.esc(note.ID)
	v.raw(`">`)
	postURL := ""
	if note.Blog != nil {
		postURL = note.Blog.Name + "/" + note.ID
	}
	a.renderPostHeader(v, data, postHeaderData{
		Blog:           note.Blog,
		Date:           note.Date,
		UseThisPostURL: postURL,
	})
	v.raw(`<div class="post-content">`)
	_, tag := a.FormatNPF(note.Content, note.Layout, note.Blog.Name, note.ID, data.RequestPollData)
	v.raw(tag)
	v.raw(`</div>`)
	if len(note.Tags) > 0 {
		v.raw(`<footer class="post-footer">`)
		renderTags(v, note.Tags, "")
		v.raw(`</footer>`)
	}
	v.raw(`</div><hr>`)
}

func (a *App) renderLikeNote(v *view, _ *PageData, note *tumblr.LikeNote) {
	avatar := ""
	if note.Avatar != nil {
		avatar = note.Avatar["128"]
	}
	v.raw(`<div class="note like"><div class="post-author">`)
	renderAvatarLink(v, note.BlogName, URLHandler(avatar))
	v.raw(`<div class="author-information"><div class="primary-post-author"><div class="blog-name-title-grouping"><a class="link blog-name" href="/`)
	v.esc(note.BlogName)
	v.raw(`">`)
	v.esc(note.BlogName)
	v.raw(`</a><span>`)
	v.esc(note.BlogTitle)
	v.raw(`</span></div></div></div></div></div>`)
}

func deseq(values url.Values) string { return values.Encode() }

func addQuery(query string) string {
	if query != "" {
		return "?" + query
	}
	return ""
}

func updateQuery(values url.Values, key, value string) string {
	copyValues := url.Values{}
	for k, v := range values {
		copyValues[k] = append([]string(nil), v...)
	}
	copyValues.Set(key, value)
	return copyValues.Encode()
}

func removeQuery(values url.Values, key string) string {
	copyValues := url.Values{}
	for k, v := range values {
		copyValues[k] = append([]string(nil), v...)
	}
	copyValues.Del(key)
	return copyValues.Encode()
}

func (a *App) renderSearchControlBar(v *view, data *PageData) {
	query := urlEscape(data.Query)
	base := data.QueryArgs
	var b strings.Builder

	// Sort-by filter.
	b.WriteString(`<li class="control-bar-action no-js" id="sort-by-filter" title="`)
	b.WriteString(htmlEscape("Sort by"))
	b.WriteString(`"><span>`)
	b.WriteString(htmlEscape(sortLabel(data.SortBy, "Popular")))
	b.WriteString(dropdownIcon)
	b.WriteString(`</span><ul class="control-bar-dropdown-menu">`)
	if data.PostFilter != "" {
		filter := urlEscape(data.PostFilter)
		b.WriteString(searchFilterItem(data.SortBy == "popular", "/search/"+query+"/"+filter+addQuery(deseq(base)), "Popular"))
		b.WriteString(searchFilterItem(data.SortBy == "recent", "/search/"+query+"/recent/"+filter+addQuery(deseq(base)), "Latest"))
	} else {
		b.WriteString(searchFilterItem(data.SortBy == "popular", "/search/"+query+addQuery(deseq(base)), "Popular"))
		b.WriteString(searchFilterItem(data.SortBy == "recent", "/search/"+query+"/recent"+addQuery(deseq(base)), "Latest"))
	}
	b.WriteString(`</ul></li>`)

	// Date filter (popular only).
	if data.SortBy == "popular" {
		b.WriteString(`<li class="control-bar-action no-js" id="filter-by-date-filter" title="`)
		b.WriteString(htmlEscape("Filter by date"))
		b.WriteString(`"><span>`)
		b.WriteString(htmlEscape(dateFilterLabel(data.TimeFilter)))
		b.WriteString(dropdownIcon)
		b.WriteString(`</span><ul class="control-bar-dropdown-menu">`)
		b.WriteString(searchFilterItem(data.TimeFilter == "0", htmlEscape(data.Path)+addQuery(removeQuery(base, "t")), "All Time"))
		for _, time := range []string{"365", "180", "30", "7", "1"} {
			b.WriteString(searchFilterItem(data.TimeFilter == time, htmlEscape(data.Path)+addQuery(updateQuery(base, "t", time)), dateFilterLabel(time)))
		}
		b.WriteString(`</ul></li>`)
	}

	// Post type filter.
	selectedLabel := "All Types"
	if data.PostFilter != "" {
		selectedLabel = postTypeLabel(data.PostFilter)
	}
	b.WriteString(`<li class="control-bar-action no-js" id="filter-by-post-type-filter" title="`)
	b.WriteString(htmlEscape("Filter by post type"))
	b.WriteString(`"><span>`)
	b.WriteString(htmlEscape(selectedLabel))
	b.WriteString(dropdownIcon)
	b.WriteString(`</span><ul class="control-bar-dropdown-menu">`)

	if data.SortBy == "recent" {
		b.WriteString(searchFilterItem(data.PostFilter == "", "/search/"+query+"/recent"+addQuery(deseq(base)), "All Types"))
	} else {
		b.WriteString(searchFilterItem(data.PostFilter == "", "/search/"+query+addQuery(deseq(base)), "All Types"))
	}
	for _, postType := range []string{"text", "photo", "gif", "quote", "link", "chat", "audio", "video", "ask", "poll"} {
		target := "/search/" + query + "/" + urlEscape(postType) + addQuery(deseq(base))
		if data.SortBy == "recent" {
			target = "/search/" + query + "/recent/" + urlEscape(postType) + addQuery(deseq(base))
		}
		b.WriteString(searchFilterItem(data.PostFilter == postType, target, postTypeLabel(postType)))
	}
	b.WriteString(`</ul></li>`)

	v.raw(b.String())
}

func searchFilterItem(selected bool, href, label string) string {
	class := ""
	if selected {
		class = ` class="selected"`
	}
	return `<li` + class + `><a href="` + href + `">` + htmlEscape(label) + `</a></li>`
}

// sortLabel returns "Latest" for recency sorting, otherwise topLabel.
func sortLabel(sortBy, topLabel string) string {
	if sortBy == "recent" {
		return "Latest"
	}
	return topLabel
}

func dateFilterLabel(filter string) string {
	switch filter {
	case "1":
		return "Today"
	case "7":
		return "Last week"
	case "30":
		return "Last month"
	case "180":
		return "Last 6 months"
	case "365":
		return "Last year"
	}
	return "All Time"
}

func postTypeLabel(postType string) string {
	switch postType {
	case "text":
		return "Text"
	case "photo":
		return "Photo"
	case "gif":
		return "Gifs"
	case "quote":
		return "Quote"
	case "link":
		return "Link"
	case "chat":
		return "Chat"
	case "audio":
		return "Audio"
	case "video":
		return "Video"
	case "ask":
		return "Ask"
	case "poll":
		return "Poll"
	}
	return "All Types"
}

func (a *App) renderTaggedControlBar(v *view, data *PageData) {
	base := data.QueryArgs
	tag := urlEscape(data.Tag)

	v.raw(`<li class="control-bar-action no-js" id="sort-by-filter" title="`)
	v.esc("Sort by")
	v.raw(`"><span>`)
	v.esc(sortLabel(data.SortBy, "Top"))
	v.raw(dropdownIcon)
	v.raw(`</span><ul class="control-bar-dropdown-menu">`)
	v.raw(searchFilterItem(data.SortBy == "top", "/tagged/"+tag+addQuery(updateQuery(base, "sort", "top")), "Top"))
	v.raw(searchFilterItem(data.SortBy == "recent", "/tagged/"+tag+addQuery(updateQuery(base, "sort", "recent")), "Latest"))
	v.raw(`</ul></li>`)
}

func (a *App) renderSearchPaging(v *view, data *PageData) {
	if data.Timeline == nil || data.Timeline.Next == "" {
		return
	}
	href := htmlEscape(data.Path) + addQuery(updateQuery(data.QueryArgs, "continuation", data.Timeline.Next))
	v.raw(`<a class="primary next-page button" href="`)
	v.raw(href)
	v.raw(`#m">`)
	v.esc("Next page")
	v.raw(`</a>`)
}
