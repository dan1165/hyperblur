package web

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	json "github.com/goccy/go-json"

	"github.com/dan1165/hyperblur/tumblr"
)

const (
	postsPerPage = 20
	// maxBlogPage bounds ?page=, so a crafted number cannot ask Tumblr for a
	// multi-million-post offset.
	maxBlogPage = 1000
)

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

	continuation := r.URL.Query().Get("continuation")
	ctx := r.Context()

	var raw map[string]any
	var err error
	switch target {
	case exploreTrending:
		raw, err = a.API.ExploreTrending(ctx, continuation)
		data.Title = "Trending topics"
		data.Endpoint = "trending"
	case exploreToday:
		raw, err = a.API.ExploreToday(ctx, continuation)
		data.Title = "Today on Tumblr"
		data.Endpoint = "today"
	default:
		raw, err = a.API.ExplorePost(ctx, postType, continuation)
		data.Title = "Trending topics"
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

	a.renderTimelinePage(w, r, data, nil, func(v *view) {
		a.renderNextPagePaging(v, data)
	})
}

// renderTimelinePage renders timeline page with control bar and paging blocks.
func (a *App) renderTimelinePage(w http.ResponseWriter, r *http.Request, data *PageData, controlBar, paging func(v *view)) {
	html := a.renderPage(data, func(v *view) {
		a.renderTimelineCenter(v, data, controlBar, paging)
	})
	a.writePage(w, r, http.StatusOK, html)
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
	v.esc("Next page")
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
	query := r.PathValue("query")
	timeFilter := normalizeTimeFilter(r.URL.Query().Get("t"))
	timeline, err := a.querySearch(r, query, timeFilter, nil, false)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	a.renderSearch(w, r, timeline, query, "popular", "", timeFilter)
}

func (a *App) handleSearchRecent(w http.ResponseWriter, r *http.Request) {
	query := r.PathValue("query")
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
	query := r.PathValue("query")
	postFilterRaw := r.PathValue("post_filter")
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
	continuation := r.URL.Query().Get("continuation")
	days, _ := strconv.Atoi(timeFilter)

	postFilter := tumblr.PostTypeFilter("")
	if filter != nil {
		postFilter = *filter
	}

	raw, err := a.API.TimelineSearch(r.Context(), query, continuation, latest, days, postFilter)
	if err != nil {
		return nil, err
	}
	return tumblr.ParseTimeline(responseOf(raw)), nil
}

func (a *App) renderSearch(w http.ResponseWriter, r *http.Request, timeline *tumblr.Timeline, query, sortBy, postFilter, timeFilter string) {
	data := a.newPageData(r)
	data.Title = query
	data.Query = query
	data.Timeline = timeline
	data.SortBy = sortBy
	data.PostFilter = postFilter
	data.TimeFilter = timeFilter
	data.QueryArgs = r.URL.Query()

	a.renderTimelinePage(w, r, data, func(v *view) {
		a.renderSearchControlBar(v, data)
	}, func(v *view) {
		a.renderSearchPaging(v, data)
	})
}

// ---------------------------------------------------------------------------
// Tagged
// ---------------------------------------------------------------------------

func (a *App) handleTagged(w http.ResponseWriter, r *http.Request) {
	tag := r.PathValue("tag")
	sortBy := r.URL.Query().Get("sort")
	latest := false
	if sortBy == "recent" {
		latest = true
	} else {
		sortBy = "top"
	}

	continuation := r.URL.Query().Get("continuation")
	raw, err := a.API.HubsTimeline(r.Context(), tag, continuation, latest)
	if err != nil {
		a.fail(w, r, err)
		return
	}

	data := a.newPageData(r)
	data.Title = tag
	data.Tag = tag
	data.SortBy = sortBy
	data.Timeline = tumblr.ParseTimeline(responseOf(raw))
	data.QueryArgs = r.URL.Query()

	a.renderTimelinePage(w, r, data, func(v *view) {
		a.renderTaggedControlBar(v, data)
	}, func(v *view) {
		a.renderNextPagePaging(v, data)
	})
}

// ---------------------------------------------------------------------------
// Blog pages
// ---------------------------------------------------------------------------

func (a *App) handleBlogIndex(w http.ResponseWriter, r *http.Request) {
	blog := r.PathValue("blog")
	if !validBlogName(blog) {
		http.NotFound(w, r)
		return
	}

	page := 1
	if raw := r.URL.Query().Get("page"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 1 {
			page = min(parsed, maxBlogPage)
		}
	}

	timeline, err := a.getBlogPosts(r.Context(), blog, "", "", strconv.Itoa((page-1)*postsPerPage), strconv.Itoa(postsPerPage))
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
	blog := r.PathValue("blog")
	tag := r.PathValue("tag")
	if !validBlogName(blog) {
		http.NotFound(w, r)
		return
	}
	continuation := r.URL.Query().Get("continuation")

	timeline, err := a.getBlogPosts(r.Context(), blog, continuation, tag, "", "")
	if err != nil {
		a.fail(w, r, err)
		return
	}

	a.renderBlogPage(w, r, blogRender{timeline: timeline, title: blog, tag: tag})
}

func (a *App) handleBlogSearchRedirect(w http.ResponseWriter, r *http.Request) {
	blog := r.PathValue("blog")
	if !validBlogName(blog) {
		http.NotFound(w, r)
		return
	}
	if query := r.URL.Query().Get("q"); query != "" {
		http.Redirect(w, r, "/"+blog+"/search/"+urlEscape(query), http.StatusFound)
		return
	}
	http.Redirect(w, r, "/"+blog, http.StatusFound)
}

func (a *App) handleBlogSearch(w http.ResponseWriter, r *http.Request) {
	blog := r.PathValue("blog")
	query := r.PathValue("query")
	if !validBlogName(blog) {
		http.NotFound(w, r)
		return
	}
	continuation := r.URL.Query().Get("continuation")
	ctx := r.Context()

	raw, err := a.API.BlogSearch(ctx, blog, query, continuation)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	timeline := tumblr.ParseBlogTimeline(responseOf(raw), true)
	if timeline == nil || timeline.BlogInfo == nil {
		// No results: fall back to blog info with empty post list.
		fallback, ferr := a.getBlogPosts(ctx, blog, "", "", "", "")
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
	a.serveBlogPost(w, r, r.PathValue("slug"))
}

func (a *App) serveBlogPost(w http.ResponseWriter, r *http.Request, slug string) {
	blog := r.PathValue("blog")
	postID := r.PathValue("post_id")
	if !validBlogName(blog) || !numericPostID(postID) {
		http.NotFound(w, r)
		return
	}

	post, err := a.getBlogPost(r.Context(), blog, postID)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	if post == nil {
		http.NotFound(w, r)
		return
	}

	// Ensure URL slug matches; redirect to canonical URL.
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
			v.esc("This blog requires an account to view")
			v.raw(`</h4><p>`)
			v.esc("Only individual posts like this one can be displayed")
			v.raw(`</p></figcaption></figure></div>`)
		}
		v.raw(`<div class="blog-posts">`)
		a.renderPost(v, data, post)
		v.raw(`</div>`)
	})
	a.writePage(w, r, http.StatusOK, html)
}

func (a *App) handleRedirectPostNoSlug(w http.ResponseWriter, r *http.Request) {
	blog := r.PathValue("blog")
	if !validBlogName(blog) {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/"+blog+"/"+r.PathValue("post_id"), http.StatusFound)
}

func (a *App) handleRedirectPost(w http.ResponseWriter, r *http.Request) {
	blog := r.PathValue("blog")
	if !validBlogName(blog) {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/"+blog+"/"+r.PathValue("post_id")+"/"+urlEscape(r.PathValue("slug")), http.StatusFound)
}

// serveNotesViewer renders replies/reblogs/likes viewer for post.
func (a *App) serveNotesViewer(w http.ResponseWriter, r *http.Request, post *tumblr.Post, noteType string) {
	blog := post.Blog.Name
	postID := post.ID
	args := r.URL.Query()
	ctx := r.Context()

	latest := args.Has("latest")
	reblogFilter := ""

	raw, err := func() (map[string]any, error) {
		switch noteType {
		case "replies":
			return a.API.BlogPostReplies(ctx, blog, postID, args.Get("after"), latest)
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
				return a.API.BlogNotes(ctx, blog, postID, true, false, before)
			}
			return a.API.BlogPostNotesTimeline(ctx, blog, postID, mode, false, before)
		default: // likes
			return a.API.BlogNotes(ctx, blog, postID, true, true, args.Get("before_timestamp"))
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
	a.writePage(w, r, http.StatusOK, html)
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
		v.esc("Search posts")
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
			v.esc("Next page")
			v.raw(`</a></div>`)
		}
	})
	a.writePage(w, r, http.StatusOK, html)
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
// API
// ---------------------------------------------------------------------------

// validPollID accepts the character set Tumblr uses for poll and answer client
// ids, which keeps "/" and ".." out of the upstream URL.
func validPollID(id string) bool {
	if id == "" || len(id) > 64 {
		return false
	}
	for i := 0; i < len(id); i++ {
		c := id[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '-', c == '_':
		default:
			return false
		}
	}
	return true
}

// writeJSONError keeps API failures JSON, since the poll viewer parses them.
func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("content-type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (a *App) handleAPIPollResults(w http.ResponseWriter, r *http.Request) {
	blog := r.PathValue("blog")
	postID := r.PathValue("post_id")
	pollID := r.PathValue("poll_id")

	if blog == "" || !validBlogName(blog) || !numericPostID(postID) || !validPollID(pollID) {
		writeJSONError(w, http.StatusBadRequest, "invalid poll reference")
		return
	}

	// Poll results are fetched live by the client; a very short cache keeps that
	// cheap without showing stale results.
	key := blog + "/" + postID + "/" + pollID
	if a.polls != nil {
		if body, ok := a.polls.Get(key); ok {
			w.Header().Set("content-type", "application/json; charset=utf-8")
			_, _ = w.Write(body)
			return
		}
	}

	ctx := r.Context()
	value, err, _ := a.pollsSF.Do(key, func() (any, error) {
		if a.polls != nil {
			if body, ok := a.polls.Get(key); ok {
				return body, nil
			}
		}
		result, err := a.API.PollResults(ctx, blog, postID, pollID)
		if err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		_ = json.NewEncoder(&buf).Encode(result["response"])
		body := buf.Bytes()
		if a.polls != nil {
			a.polls.Add(key, body)
		}
		return body, nil
	})
	if err != nil {
		a.Logger.Printf("poll results for %s: %v", key, err)
		writeJSONError(w, http.StatusBadGateway, "could not load poll results")
		return
	}
	body, _ := value.([]byte)

	w.Header().Set("content-type", "application/json; charset=utf-8")
	_, _ = w.Write(body)
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func responseOf(raw map[string]any) map[string]any {
	response, _ := raw["response"].(map[string]any)
	return response
}

func (a *App) getBlogPosts(ctx context.Context, blog, continuation, tag, offset, limit string) (*tumblr.BlogTimeline, error) {
	raw, err := a.API.BlogPosts(ctx, blog, continuation, tag, offset, limit)
	if err != nil {
		return nil, err
	}
	timeline := tumblr.ParseBlogTimeline(responseOf(raw), false)
	if timeline == nil {
		return &tumblr.BlogTimeline{BlogInfo: &tumblr.Blog{Name: blog, Active: false}, Posts: []*tumblr.Post{}}, nil
	}
	return timeline, nil
}

func (a *App) getBlogPost(ctx context.Context, blog, postID string) (*tumblr.Post, error) {
	raw, err := a.API.BlogPost(ctx, blog, postID)
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
