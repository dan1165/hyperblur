package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/dan1165/hyperblur/tumblr"
)

const postsPerPage = 20

type exploreTarget int

const (
	exploreTrending exploreTarget = iota
	exploreToday
	explorePost
)

// ---------------------------------------------------------------------------
// Explore
// ---------------------------------------------------------------------------

func (a *App) handleExploreIndex(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/explore/trending", http.StatusFound)
}

func (a *App) handleExplore(w http.ResponseWriter, r *http.Request, target exploreTarget, postType tumblr.ExplorePostType) {
	data := a.newPageData(r)
	data.ExtraCSS = []string{"/assets/css/timeline.css"}

	continuation := unquoteQuery(r.URL.Query().Get("continuation"))

	var raw map[string]any
	var err error
	switch target {
	case exploreTrending:
		raw, err = a.API.ExploreTrending(continuation)
		data.Title = a.translate("explore_trending_page_title")
		data.Endpoint = "trending"
	case exploreToday:
		raw, err = a.API.ExploreToday(continuation)
		data.Title = a.translate("explore_today_on_tumblr_page_title")
		data.Endpoint = "today"
	default:
		raw, err = a.API.ExplorePost(postType, continuation)
		data.Title = a.translate("explore_trending_page_title")
	}
	if err != nil {
		a.fail(w, r, err)
		return
	}
	data.Timeline = tumblr.ParseTimeline(responseOf(raw))
	if data.Timeline == nil {
		a.genericError(w, r, errors.New("unexpected empty explore response"))
		return
	}

	html := a.renderPage(data, func(v *view) {
		a.renderTimelineCenter(v, data, nil, func(v *view) {
			a.renderNextPagePaging(v, data)
		})
	})
	a.writePage(w, http.StatusOK, html)
}

func (a *App) renderNextPagePaging(v *view, data *PageData) {
	if data.Timeline == nil || data.Timeline.Next == "" {
		return
	}
	v.raw(`<a class="primary next-page button" href="`)
	v.esc(data.Path)
	v.raw(`?continuation=`)
	v.esc(url.QueryEscape(data.Timeline.Next))
	v.raw(`#m">`)
	v.esc(a.translate("pagination_next_page"))
	v.raw(`</a>`)
}

// ---------------------------------------------------------------------------
// Search
// ---------------------------------------------------------------------------

func (a *App) handleSearchRedirect(w http.ResponseWriter, r *http.Request) {
	if query := r.URL.Query().Get("q"); query != "" {
		http.Redirect(w, r, "/search/"+urlEscape(query), http.StatusFound)
		return
	}
	http.Redirect(w, r, "/explore/trending", http.StatusFound)
}

func (a *App) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := unquotePath(r.PathValue("query"))
	timeFilter := normalizeTimeFilter(r.URL.Query().Get("t"))
	timeline, err := a.querySearch(r, query, timeFilter, nil, false)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	a.renderSearch(w, r, timeline, query, "popular", "", timeFilter)
}

func (a *App) handleSearchRecent(w http.ResponseWriter, r *http.Request) {
	query := unquotePath(r.PathValue("query"))
	timeFilter := normalizeTimeFilter(r.URL.Query().Get("t"))
	timeline, err := a.querySearch(r, query, timeFilter, nil, true)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	a.renderSearch(w, r, timeline, query, "recent", "", timeFilter)
}

func (a *App) handleSearchFilter(w http.ResponseWriter, r *http.Request) {
	a.searchFilter(w, r, false)
}

func (a *App) handleSearchRecentFilter(w http.ResponseWriter, r *http.Request) {
	a.searchFilter(w, r, true)
}

func (a *App) searchFilter(w http.ResponseWriter, r *http.Request, latest bool) {
	query := unquotePath(r.PathValue("query"))
	postFilterRaw := unquotePath(r.PathValue("post_filter"))
	timeFilter := normalizeTimeFilter(r.URL.Query().Get("t"))

	filter := parsePostFilter(postFilterRaw)
	if filter == "" {
		target := "/search/" + urlEscape(query)
		if latest {
			target = "/search/" + urlEscape(query) + "/recent"
		}
		if r.URL.RawQuery != "" {
			target += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, target, http.StatusFound)
		return
	}

	timeline, err := a.querySearch(r, query, timeFilter, &filter, latest)
	if err != nil {
		a.fail(w, r, err)
		return
	}

	displayFilter := string(filter)
	sortBy := "popular"
	if latest {
		sortBy = "recent"
	}
	a.renderSearch(w, r, timeline, query, sortBy, displayFilter, timeFilter)
}

func (a *App) querySearch(r *http.Request, query, timeFilter string, filter *tumblr.PostTypeFilter, latest bool) (*tumblr.Timeline, error) {
	continuation := unquoteQuery(r.URL.Query().Get("continuation"))
	days, _ := strconv.Atoi(timeFilter)

	postFilter := tumblr.PostTypeFilter("")
	if filter != nil {
		postFilter = *filter
	}

	raw, err := a.API.TimelineSearch(query, continuation, latest, days, postFilter)
	if err != nil {
		return nil, err
	}
	return tumblr.ParseTimeline(responseOf(raw)), nil
}

func (a *App) renderSearch(w http.ResponseWriter, r *http.Request, timeline *tumblr.Timeline, query, sortBy, postFilter, timeFilter string) {
	data := a.newPageData(r)
	data.ExtraCSS = []string{"/assets/css/timeline.css"}
	data.Title = query
	data.Query = query
	data.Timeline = timeline
	data.SortBy = sortBy
	data.PostFilter = postFilter
	data.TimeFilter = timeFilter
	data.QueryArgs = r.URL.Query()

	html := a.renderPage(data, func(v *view) {
		a.renderTimelineCenter(v, data, func(v *view) {
			a.renderSearchControlBar(v, data)
		}, func(v *view) {
			a.renderSearchPaging(v, data)
		})
	})
	a.writePage(w, http.StatusOK, html)
}

// ---------------------------------------------------------------------------
// Tagged
// ---------------------------------------------------------------------------

func (a *App) handleTagged(w http.ResponseWriter, r *http.Request) {
	tag := unquotePath(r.PathValue("tag"))
	sortBy := r.URL.Query().Get("sort")
	latest := false
	if sortBy == "recent" {
		latest = true
	} else {
		sortBy = "top"
	}

	continuation := unquoteQuery(r.URL.Query().Get("continuation"))
	raw, err := a.API.HubsTimeline(tag, continuation, latest)
	if err != nil {
		a.fail(w, r, err)
		return
	}

	data := a.newPageData(r)
	data.ExtraCSS = []string{"/assets/css/timeline.css"}
	data.Title = tag
	data.Tag = tag
	data.SortBy = sortBy
	data.Timeline = tumblr.ParseTimeline(responseOf(raw))
	data.QueryArgs = r.URL.Query()

	html := a.renderPage(data, func(v *view) {
		a.renderTimelineCenter(v, data, func(v *view) {
			a.renderTaggedControlBar(v, data)
		}, func(v *view) {
			a.renderNextPagePaging(v, data)
		})
	})
	a.writePage(w, http.StatusOK, html)
}

// ---------------------------------------------------------------------------
// Blog pages
// ---------------------------------------------------------------------------

func (a *App) handleBlogIndex(w http.ResponseWriter, r *http.Request) {
	blog := unquotePath(r.PathValue("blog"))
	if !validBlogName(blog) {
		http.NotFound(w, r)
		return
	}

	page := 1
	if raw := r.URL.Query().Get("page"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 1 {
			page = parsed
		}
	}

	timeline, err := a.getBlogPosts(blog, "", "", strconv.Itoa((page-1)*postsPerPage), strconv.Itoa(postsPerPage))
	if err != nil {
		a.fail(w, r, err)
		return
	}

	total := 0
	if timeline.TotalPosts != nil {
		total = *timeline.TotalPosts
	}
	totalPages := 1
	if total > 0 {
		totalPages = (total + postsPerPage - 1) / postsPerPage
	}

	visible := map[int]bool{1: true, totalPages: true}
	for p := page - 2; p <= page+2; p++ {
		if p >= 1 && p <= totalPages {
			visible[p] = true
		}
	}
	var pageNumbers []int
	previous := 0
	for p := 1; p <= totalPages; p++ {
		if !visible[p] {
			continue
		}
		if previous != 0 && p-previous > 1 {
			pageNumbers = append(pageNumbers, 0)
		}
		pageNumbers = append(pageNumbers, p)
		previous = p
	}

	a.renderBlogPage(w, r, blogRender{
		timeline:    timeline,
		title:       blog,
		page:        page,
		pageNumbers: pageNumbers,
	})
}

func (a *App) handleBlogTags(w http.ResponseWriter, r *http.Request) {
	blog := unquotePath(r.PathValue("blog"))
	tag := unquotePath(r.PathValue("tag"))
	if !validBlogName(blog) {
		http.NotFound(w, r)
		return
	}
	continuation := unquoteQuery(r.URL.Query().Get("continuation"))

	timeline, err := a.getBlogPosts(blog, continuation, tag, "", "")
	if err != nil {
		a.fail(w, r, err)
		return
	}

	a.renderBlogPage(w, r, blogRender{timeline: timeline, title: blog, tag: tag})
}

func (a *App) handleBlogSearchRedirect(w http.ResponseWriter, r *http.Request) {
	blog := unquotePath(r.PathValue("blog"))
	if query := r.URL.Query().Get("q"); query != "" {
		http.Redirect(w, r, "/"+blog+"/search/"+urlEscape(query), http.StatusFound)
		return
	}
	http.Redirect(w, r, "/"+blog, http.StatusFound)
}

func (a *App) handleBlogSearch(w http.ResponseWriter, r *http.Request) {
	blog := unquotePath(r.PathValue("blog"))
	query := unquotePath(r.PathValue("query"))
	if !validBlogName(blog) {
		http.NotFound(w, r)
		return
	}
	continuation := unquoteQuery(r.URL.Query().Get("continuation"))

	raw, err := a.API.BlogSearch(blog, query, continuation)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	timeline := tumblr.ParseBlogTimeline(responseOf(raw), true)
	if timeline == nil || timeline.BlogInfo == nil {
		// No results: fall back to the blog's own info with an empty post list.
		fallback, ferr := a.getBlogPosts(blog, "", "", "", "")
		if ferr != nil {
			a.fail(w, r, ferr)
			return
		}
		timeline = &tumblr.BlogTimeline{BlogInfo: fallback.BlogInfo, Posts: []*tumblr.Post{}, TotalPosts: intPtr(0)}
	}

	a.renderBlogPage(w, r, blogRender{timeline: timeline, title: blog, searchQuery: query})
}

func (a *App) handleBlogPost(w http.ResponseWriter, r *http.Request) {
	a.serveBlogPost(w, r, "")
}

func (a *App) handleBlogPostWithSlug(w http.ResponseWriter, r *http.Request) {
	a.serveBlogPost(w, r, unquotePath(r.PathValue("slug")))
}

func (a *App) serveBlogPost(w http.ResponseWriter, r *http.Request, slug string) {
	blog := unquotePath(r.PathValue("blog"))
	postID := r.PathValue("post_id")
	if !validBlogName(blog) || !numericPostID(postID) {
		http.NotFound(w, r)
		return
	}

	post, err := a.getBlogPost(blog, postID)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	if post == nil {
		http.NotFound(w, r)
		return
	}

	// Ensure the slug in the URL matches (redirect to the canonical URL).
	if slug != post.Slug {
		path := "/" + PostPath(post)
		if post.Slug == "" {
			path = "/" + post.Blog.Name + "/" + post.ID
		}
		if r.URL.RawQuery != "" {
			path += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, path, http.StatusFound)
		return
	}

	timeline := &tumblr.BlogTimeline{BlogInfo: post.Blog, Posts: []*tumblr.Post{}}

	if noteType := strings.ToLower(r.URL.Query().Get("note_viewer")); noteType == "replies" || noteType == "reblogs" || noteType == "likes" {
		a.serveNotesViewer(w, r, post, noteType)
		return
	}

	data := a.newPageData(r)
	data.ExtraCSS = []string{"/assets/css/blog.css"}
	data.Title = post.Blog.Title
	if data.Title == "" {
		data.Title = post.Blog.Name
	}
	data.Blog = timeline
	data.PostURL = strings.TrimPrefix(r.URL.Path, "/")
	data.BlogTagName = post.Blog.Name
	data.RequestPollData = truthyQuery(r.URL.Query().Get("fetch_polls"))

	html := a.renderPage(data, func(v *view) {
		a.renderBlogHeader(v, data)
		if post.Blog.RequiresAccountToView {
			v.raw(`<div class="alerts"><figure class="alert warning"><figcaption><h4>`)
			v.esc(a.translate("alert_view_post_on_account_restricted_blog_heading"))
			v.raw(`</h4><p>`)
			v.esc(a.translate("alert_view_post_on_account_restricted_blog_message"))
			v.raw(`</p></figcaption></figure></div>`)
		}
		v.raw(`<div class="blog-posts">`)
		a.renderPost(v, data, post)
		v.raw(`</div>`)
	})
	a.writePage(w, http.StatusOK, html)
}

func (a *App) handleRedirectPostNoSlug(w http.ResponseWriter, r *http.Request) {
	blog := unquotePath(r.PathValue("blog"))
	postID := r.PathValue("post_id")
	http.Redirect(w, r, "/"+blog+"/"+postID, http.StatusFound)
}

func (a *App) handleRedirectPost(w http.ResponseWriter, r *http.Request) {
	blog := unquotePath(r.PathValue("blog"))
	postID := r.PathValue("post_id")
	slug := unquotePath(r.PathValue("slug"))
	http.Redirect(w, r, "/"+blog+"/"+postID+"/"+slug, http.StatusFound)
}

// serveNotesViewer renders the replies/reblogs/likes note viewer for a post.
func (a *App) serveNotesViewer(w http.ResponseWriter, r *http.Request, post *tumblr.Post, noteType string) {
	blog := post.Blog.Name
	postID := post.ID
	args := r.URL.Query()

	latest := args.Has("latest")
	reblogFilter := ""

	raw, err := func() (map[string]any, error) {
		switch noteType {
		case "replies":
			return a.API.BlogPostReplies(blog, postID, args.Get("after"), latest)
		case "reblogs":
			mode := tumblr.ReblogsWithComments
			switch args.Get("reblog_filter") {
			case "reblogs_with_comments":
				mode = tumblr.ReblogsWithComments
				reblogFilter = "reblogs_with_comments"
			case "reblogs_with_content_comments":
				mode = tumblr.ReblogsWithContentComments
				reblogFilter = "reblogs_with_content_comments"
			case "reblogs_only":
				mode = tumblr.ReblogsOnly
				reblogFilter = "reblogs_only"
			}
			before := args.Get("before_timestamp")
			if mode == tumblr.ReblogsOnly {
				return a.API.BlogNotes(blog, postID, true, false, before)
			}
			return a.API.BlogPostNotesTimeline(blog, postID, mode, false, before)
		default: // likes
			return a.API.BlogNotes(blog, postID, true, true, args.Get("before_timestamp"))
		}
	}()
	if err != nil {
		a.fail(w, r, err)
		return
	}

	data := a.newPageData(r)
	data.Title = post.Blog.Title
	if data.Title == "" {
		data.Title = post.Blog.Name
	}
	data.PostURL = strings.TrimPrefix(r.URL.Path, "/")
	data.Notes = tumblr.ParseNoteTimeline(responseOf(raw))
	data.NoteType = noteType
	data.Latest = latest
	data.ReblogFilter = reblogFilter
	data.QueryArgs = args

	html := a.renderPage(data, func(v *view) {
		a.renderNotesViewer(v, data)
	})
	a.writePage(w, http.StatusOK, html)
}

type blogRender struct {
	timeline    *tumblr.BlogTimeline
	title       string
	tag         string
	page        int
	pageNumbers []int
	searchQuery string
}

func (a *App) renderBlogPage(w http.ResponseWriter, r *http.Request, br blogRender) {
	data := a.newPageData(r)
	data.ExtraCSS = []string{"/assets/css/blog.css"}
	data.Title = br.title
	data.Blog = br.timeline
	data.Tag = br.tag
	data.BlogTagName = br.timeline.BlogInfo.Name
	data.BlogSearchQuery = br.searchQuery
	data.Page = br.page
	data.PageNumbers = br.pageNumbers

	html := a.renderPage(data, func(v *view) {
		a.renderBlogHeader(v, data)
		v.raw(`<form class="blog-search-bar search-bar" method="get" action="/`)
		v.esc(br.timeline.BlogInfo.Name)
		v.raw(`/search" autocomplete="off"><svg class="icon" xmlns="http://www.w3.org/2000/svg" height="20" viewBox="0 -960 960 960" width="20"><path d="M765-144 526-383q-30 22-65.792 34.5T384.035-336Q284-336 214-406t-70-170q0-100 70-170t170-70q100 0 170 70t70 170.035q0 40.381-12.5 76.173T577-434l239 239-51 51ZM384-408q70 0 119-49t49-119q0-70-49-119t-119-49q-70 0-119 49t-49 119q0 70 49 119t119 49Z"/></svg><input name="q" type="text"`)
		if br.searchQuery != "" {
			v.raw(` value="`)
			v.esc(br.searchQuery)
			v.raw(`"`)
		}
		v.raw(` placeholder="`)
		v.esc(a.translate("blog_search_placeholder_text"))
		v.raw(`"></input></form>`)

		v.raw(`<div class="blog-posts">`)
		for _, post := range br.timeline.Posts {
			if !post.IsAdvertisement {
				a.renderPost(v, data, post)
			}
		}
		v.raw(`</div>`)

		if len(br.pageNumbers) > 1 {
			a.renderNumberedPaging(v, data)
		} else if br.timeline.Next != "" {
			v.raw(`<div class="paging"><a class="primary next-page button" href="`)
			v.esc(r.URL.Path)
			v.raw(`?continuation=`)
			v.esc(url.QueryEscape(br.timeline.Next))
			v.raw(`#m">`)
			v.esc(a.translate("pagination_next_page"))
			v.raw(`</a></div>`)
		}
	})
	a.writePage(w, http.StatusOK, html)
}

func (a *App) renderNumberedPaging(v *view, data *PageData) {
	v.raw(`<div class="paging">`)
	if data.Page > 1 {
		v.raw(`<a class="page-link" href="`)
		v.esc(data.Path)
		v.raw(`?page=`)
		v.raw(strconv.Itoa(data.Page - 1))
		v.raw(`#m">‹</a>`)
	}
	for _, p := range data.PageNumbers {
		switch {
		case p == 0:
			v.raw(`<span class="page-link ellipsis">…</span>`)
		case p == data.Page:
			v.raw(`<span class="page-link current">`)
			v.raw(strconv.Itoa(p))
			v.raw(`</span>`)
		default:
			v.raw(`<a class="page-link" href="`)
			v.esc(data.Path)
			v.raw(`?page=`)
			v.raw(strconv.Itoa(p))
			v.raw(`#m">`)
			v.raw(strconv.Itoa(p))
			v.raw(`</a>`)
		}
	}
	if len(data.PageNumbers) > 0 && data.Page < data.PageNumbers[len(data.PageNumbers)-1] {
		v.raw(`<a class="page-link" href="`)
		v.esc(data.Path)
		v.raw(`?page=`)
		v.raw(strconv.Itoa(data.Page + 1))
		v.raw(`#m">›</a>`)
	}
	v.raw(`</div>`)
}

// ---------------------------------------------------------------------------
// Settings
// ---------------------------------------------------------------------------

func (a *App) handleSettings(w http.ResponseWriter, r *http.Request) {
	a.renderSettings(w, r, nil)
}

func (a *App) handleSettingsPost(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	prefs := preferencesFrom(r)
	prefs.ExpandPosts = r.FormValue("expand_posts") == "on"
	a.renderSettings(w, r, &prefs)
}

func (a *App) handleSettingsRestore(w http.ResponseWriter, r *http.Request) {
	prefs := preferencesFrom(r)
	if value := r.URL.Query().Get("expand_posts"); value != "" {
		prefs.ExpandPosts = value == "on"
	}
	a.renderSettings(w, r, &prefs)
}

func (a *App) renderSettings(w http.ResponseWriter, r *http.Request, newPrefs *Preferences) {
	prefs := preferencesFrom(r)
	if newPrefs != nil {
		prefs = *newPrefs
	}

	data := a.newPageData(r)
	data.ExpandPosts = prefs.ExpandPosts
	data.Title = a.translate("settings_header")
	data.ExtraCSS = []string{"/assets/css/settings.css"}

	html := a.renderPage(data, func(v *view) {
		v.raw(`<form class="settings" method="post" action="/settings" aria-label="`)
		v.esc(a.translate("settings_header"))
		v.raw(`"><div id="setting-heading"><h2>`)
		v.esc(a.translate("settings_header"))
		v.raw(`</h2></div><hr><div class="main-tab"><div id="expand-posts-option" aria-describedby="expand-posts-info-box"><div class="option-info-box"><label for="expand-posts-checkbox">`)
		v.esc(a.translate("settings_expand_blogger_truncated_posts"))
		v.raw(`</label><p id="expand-posts-info-box">`)
		v.esc(a.translate("settings_expand_blogger_truncated_posts_desc"))
		v.raw(`</p></div><input type="checkbox" id="expand-posts-checkbox" name="expand_posts"`)
		if prefs.ExpandPosts {
			v.raw(` checked`)
		}
		v.raw(`/><input type="hidden" id="expand-posts-checkbox" name="expand_posts" value="off"/></div></div>`)

		v.raw(`<div id="settings-footer"><div id="copy-as-bookmarklet-container"><a id="copy-as-bookmarklet" href="/settings/restore?`)
		v.raw(preferencesToURL(prefs))
		v.raw(`"><svg height="12" width="12" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 640 512"><path fill="currentColor" d="M579.8 267.7c56.5-56.5 56.5-148 0-204.5c-50-50-128.8-56.5-186.3-15.4l-1.6 1.1c-14.4 10.3-17.7 30.3-7.4 44.6s30.3 17.7 44.6 7.4l1.6-1.1c32.1-22.9 76-19.3 103.8 8.6c31.5 31.5 31.5 82.5 0 114L422.3 334.8c-31.5 31.5-82.5 31.5-114 0c-27.9-27.9-31.5-71.8-8.6-103.8l1.1-1.6c10.3-14.4 6.9-34.4-7.4-44.6s-34.4-6.9-44.6 7.4l-1.1 1.6C206.5 251.2 213 330 263 380c56.5 56.5 148 56.5 204.5 0L579.8 267.7zM60.2 244.3c-56.5 56.5-56.5 148 0 204.5c50 50 128.8 56.5 186.3 15.4l1.6-1.1c14.4-10.3 17.7-30.3 7.4-44.6s-30.3-17.7-44.6-7.4l-1.6 1.1c-32.1 22.9-76 19.3-103.8-8.6C74 372 74 321 105.5 289.5L217.7 177.2c31.5-31.5 82.5-31.5 114 0c27.9 27.9 31.5 71.8 8.6 103.9l-1.1 1.6c-10.3 14.4-6.9 34.4 7.4 44.6s34.4 6.9 44.6-7.4l1.1-1.6C433.5 260.8 427 182 377 132c-56.5-56.5-148-56.5-204.5 0L60.2 244.3z"/></svg><svg style="display: none;" height="12" width="12" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 448 512"><path fill="currentColor" d="M438.6 105.4c12.5 12.5 12.5 32.8 0 45.3l-256 256c-12.5 12.5-32.8 12.5-45.3 0l-128-128c-12.5-12.5-12.5-32.8 0-45.3s32.8-12.5 45.3 0L160 338.7 393.4 105.4c12.5-12.5 32.8-12.5 45.3 0z"/></svg><svg style="display: none;" height="12" width="12" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 384 512"><path fill="currentColor" d="M342.6 150.6c12.5-12.5 12.5-32.8 0-45.3s-32.8-12.5-45.3 0L192 210.7 86.6 105.4c-12.5-12.5-32.8-12.5-45.3 0s-12.5 32.8 0 45.3L146.7 256 41.4 361.4c-12.5 12.5-12.5 32.8 0 45.3s32.8 12.5 45.3 0L192 301.3 297.4 406.6c12.5 12.5 32.8 12.5 45.3 0s12.5-32.8 0-45.3L237.3 256 342.6 150.6z"/></svg><span>`)
		v.esc(a.translate("settings_copy_as_bookmarklet"))
		v.raw(`</span></a></div><div><a href="/settings" class="secondary button">`)
		v.esc(a.translate("settings_cancel_changes"))
		v.raw(`</a><input type="submit" class="primary button" value="`)
		v.esc(a.translate("settings_save_changes"))
		v.raw(`"/></div></div></form>`)
	})

	if newPrefs != nil {
		http.SetCookie(w, &http.Cookie{
			Name:   "settings",
			Value:  preferencesToURL(*newPrefs),
			MaxAge: 31540000,
			Path:   "/",
		})
	}
	a.writePage(w, http.StatusOK, html)
}

func preferencesToURL(prefs Preferences) string {
	value := "off"
	if prefs.ExpandPosts {
		value = "on"
	}
	return "expand_posts=" + value
}

// ---------------------------------------------------------------------------
// API
// ---------------------------------------------------------------------------

func (a *App) handleAPIPollResults(w http.ResponseWriter, r *http.Request) {
	blog := unquotePath(r.PathValue("blog"))
	postID := r.PathValue("post_id")
	pollID := unquotePath(r.PathValue("poll_id"))

	result, err := a.API.PollResults(blog, postID, pollID)
	if err != nil {
		a.fail(w, r, err)
		return
	}

	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(result["response"])
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func responseOf(raw map[string]any) map[string]any {
	response, _ := raw["response"].(map[string]any)
	return response
}

func (a *App) getBlogPosts(blog, continuation, tag, offset, limit string) (*tumblr.BlogTimeline, error) {
	raw, err := a.API.BlogPosts(blog, continuation, tag, offset, limit)
	if err != nil {
		return nil, err
	}
	timeline := tumblr.ParseBlogTimeline(responseOf(raw), false)
	if timeline == nil {
		return &tumblr.BlogTimeline{BlogInfo: &tumblr.Blog{Name: blog, Active: false}, Posts: []*tumblr.Post{}}, nil
	}
	return timeline, nil
}

func (a *App) getBlogPost(blog, postID string) (*tumblr.Post, error) {
	raw, err := a.API.BlogPost(blog, postID)
	if err != nil {
		return nil, err
	}
	timeline := tumblr.ParseTimeline(responseOf(raw))
	if timeline == nil || len(timeline.Elements) == 0 {
		return nil, nil
	}
	post, _ := timeline.Elements[0].(*tumblr.Post)
	return post, nil
}

func intPtr(v int) *int { return &v }

func unquotePath(s string) string {
	decoded, err := url.PathUnescape(s)
	if err != nil {
		return s
	}
	return decoded
}

func unquoteQuery(s string) string {
	decoded, err := url.QueryUnescape(s)
	if err != nil {
		return s
	}
	return decoded
}

func truthyQuery(s string) bool {
	switch strings.ToLower(s) {
	case "1", "true", "on", "yes":
		return true
	}
	return false
}

func normalizeTimeFilter(v string) string {
	switch v {
	case "365", "180", "30", "7", "1":
		return v
	}
	return "0"
}

func parsePostFilter(raw string) tumblr.PostTypeFilter {
	switch strings.ToUpper(raw) {
	case "ASK", "ANSWER":
		return tumblr.FilterAnswer
	case "TEXT":
		return tumblr.FilterText
	case "PHOTO":
		return tumblr.FilterPhoto
	case "GIF":
		return tumblr.FilterGIF
	case "QUOTE":
		return tumblr.FilterQuote
	case "LINK":
		return tumblr.FilterLink
	case "CHAT":
		return tumblr.FilterChat
	case "AUDIO":
		return tumblr.FilterAudio
	case "VIDEO":
		return tumblr.FilterVideo
	case "POLL":
		return tumblr.FilterPoll
	}
	return ""
}
