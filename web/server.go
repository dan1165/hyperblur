package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/dan1165/openblur/tumblr"
)

type contextKey int

const prefsKey contextKey = 0

// Preferences holds a visitor's settings.
type Preferences struct {
	ExpandPosts bool
}

func defaultPreferences() Preferences { return Preferences{ExpandPosts: true} }

func preferencesFromRequest(r *http.Request) Preferences {
	prefs := defaultPreferences()
	cookie, err := r.Cookie("settings")
	if err != nil {
		return prefs
	}
	values, err := url.ParseQuery(cookie.Value)
	if err != nil {
		return prefs
	}
	if value := values.Get("expand_posts"); value != "" {
		prefs.ExpandPosts = value == "on"
	}
	return prefs
}

func preferencesFrom(r *http.Request) Preferences {
	if prefs, ok := r.Context().Value(prefsKey).(Preferences); ok {
		return prefs
	}
	return defaultPreferences()
}

// Handler builds the HTTP handler for the application.
func (a *App) Handler(assetsDir string) http.Handler {
	mux := http.NewServeMux()

	assets := http.StripPrefix("/assets/", http.FileServer(http.Dir(assetsDir)))
	mux.Handle("GET /assets/", assets)

	mux.HandleFunc("GET /{$}", a.handleRoot)
	mux.HandleFunc("GET /robots.txt", a.handleRobots)

	// Media proxying.
	mux.HandleFunc("GET /tblr/media/{cdn}/{path...}", a.handleMediaCDN)
	mux.HandleFunc("GET /tblr/a/{path...}", a.handleMediaAudio)
	mux.HandleFunc("GET /tblr/assets/{path...}", a.handleMediaAssets)
	mux.HandleFunc("GET /tblr/static/{path...}", a.handleMediaStatic)

	// at.tumblr.com link redirects.
	mux.HandleFunc("GET /at/{path...}", a.handleAtLinks)

	// Settings.
	mux.HandleFunc("GET /settings", a.handleSettings)
	mux.HandleFunc("GET /settings/{$}", a.handleSettings)
	mux.HandleFunc("POST /settings/{$}", a.handleSettingsPost)
	mux.HandleFunc("GET /settings/restore", a.handleSettingsRestore)

	// API.
	mux.HandleFunc("GET /api/v1/poll/{blog}/{post_id}/{poll_id}/results", a.handleAPIPollResults)

	// Blog pages are dispatched by the fallback handler because Go's
	// ServeMux rejects the wildcard patterns they need next to /assets/.
	mux.HandleFunc("GET /", a.handleFallback)

	return a.middleware(mux)
}

// handleFallback dispatches explore/search/tagged/blog paths. They are handled
// here rather than as ServeMux patterns because the wildcard patterns they need
// conflict with each other and with /assets/.
func (a *App) handleFallback(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.Trim(r.URL.Path, "/")
	if trimmed == "" {
		a.handleRoot(w, r)
		return
	}
	segments := strings.Split(trimmed, "/")

	// A trailing /rss marks an RSS feed; strip it before dispatching.
	if len(segments) > 1 && segments[len(segments)-1] == "rss" {
		segments = segments[:len(segments)-1]
	}

	switch segments[0] {
	case "explore":
		a.routeExplore(w, r, segments)
		return
	case "search":
		a.routeSearch(w, r, segments)
		return
	case "tagged":
		a.routeTagged(w, r, segments)
		return
	}

	r.SetPathValue("blog", segments[0])
	switch {
	case len(segments) == 1:
		a.handleBlogIndex(w, r)
	case len(segments) == 2 && segments[1] == "search":
		a.handleBlogSearchRedirect(w, r)
	case len(segments) == 3 && segments[1] == "tagged":
		r.SetPathValue("tag", segments[2])
		a.handleBlogTags(w, r)
	case len(segments) == 3 && segments[1] == "search":
		r.SetPathValue("query", segments[2])
		a.handleBlogSearch(w, r)
	case len(segments) == 3 && segments[1] == "post":
		r.SetPathValue("post_id", segments[2])
		a.handleRedirectPostNoSlug(w, r)
	case len(segments) == 4 && segments[1] == "post":
		r.SetPathValue("post_id", segments[2])
		r.SetPathValue("slug", segments[3])
		a.handleRedirectPost(w, r)
	case len(segments) == 2:
		r.SetPathValue("post_id", segments[1])
		a.handleBlogPost(w, r)
	case len(segments) == 3:
		r.SetPathValue("post_id", segments[1])
		r.SetPathValue("slug", segments[2])
		a.handleBlogPostWithSlug(w, r)
	default:
		http.NotFound(w, r)
	}
}

var exploreTypes = map[string]tumblr.ExplorePostType{
	"text": tumblr.ExploreText, "photos": tumblr.ExplorePhotos, "gifs": tumblr.ExploreGIFs,
	"quotes": tumblr.ExploreQuotes, "chats": tumblr.ExploreChats, "audio": tumblr.ExploreAudio,
	"video": tumblr.ExploreVideo, "asks": tumblr.ExploreAsks,
}

func (a *App) routeExplore(w http.ResponseWriter, r *http.Request, segments []string) {
	if len(segments) == 1 {
		a.handleExploreIndex(w, r)
		return
	}
	switch segments[1] {
	case "trending":
		a.handleExplore(w, r, exploreTrending, "")
	case "today":
		a.handleExplore(w, r, exploreToday, "")
	default:
		postType, ok := exploreTypes[segments[1]]
		if !ok {
			http.NotFound(w, r)
			return
		}
		a.handleExplore(w, r, explorePost, postType)
	}
}

func (a *App) routeSearch(w http.ResponseWriter, r *http.Request, segments []string) {
	switch len(segments) {
	case 1:
		a.handleSearchRedirect(w, r)
	case 2:
		r.SetPathValue("query", segments[1])
		a.handleSearch(w, r)
	case 3:
		r.SetPathValue("query", segments[1])
		if segments[2] == "recent" {
			a.handleSearchRecent(w, r)
		} else {
			r.SetPathValue("post_filter", segments[2])
			a.handleSearchFilter(w, r)
		}
	case 4:
		if segments[2] != "recent" {
			http.NotFound(w, r)
			return
		}
		r.SetPathValue("query", segments[1])
		r.SetPathValue("post_filter", segments[3])
		a.handleSearchRecentFilter(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (a *App) routeTagged(w http.ResponseWriter, r *http.Request, segments []string) {
	if len(segments) < 2 {
		http.NotFound(w, r)
		return
	}
	r.SetPathValue("tag", segments[1])
	a.handleTagged(w, r)
}

var blogNamePattern = regexp.MustCompile(`^[a-z\d][a-z\d-]{0,30}[a-z\d]$`)

func validBlogName(name string) bool {
	return name == "" || blogNamePattern.MatchString(name)
}

func numericPostID(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

func (a *App) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		prefs := preferencesFromRequest(r)
		r = r.WithContext(context.WithValue(r.Context(), prefsKey, prefs))

		header := w.Header()
		header.Set("x-xss-protection", "1; mode=block")
		header.Set("x-content-type-options", "nosniff")
		header.Set("referrer-policy", "same-origin")
		header.Set("cache-control", "no-store, no-cache, must-revalidate, max-age=0")
		header.Set("pragma", "no-cache")
		header.Set("expires", "0")
		header.Set("content-security-policy", strings.Join([]string{
			"default-src 'none'",
			"script-src 'self'",
			"style-src 'self' 'unsafe-inline'",
			"img-src 'self' data: https://*.tumblr.com",
			"font-src 'self' data:",
			"connect-src 'self'",
			"manifest-src 'self'",
			"media-src 'self' https://*.tumblr.com",
			"child-src 'self' blob:",
		}, "; "))

		next.ServeHTTP(w, r)
	})
}

// ---------------------------------------------------------------------------
// Basic handlers
// ---------------------------------------------------------------------------

func (a *App) handleRoot(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/explore/trending", http.StatusFound)
}

func (a *App) handleRobots(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join("assets", "robots.txt")
	data, err := os.ReadFile(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("content-type", "text/plain; charset=utf-8")
	w.Write(data)
}

// ---------------------------------------------------------------------------
// Error rendering
// ---------------------------------------------------------------------------

func (a *App) writePage(w http.ResponseWriter, status int, html string) {
	w.WriteHeader(status)
	_, _ = w.Write([]byte(html))
}

func (a *App) writeRSS(w http.ResponseWriter, body string) {
	w.Header().Set("content-type", "application/rss+xml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(body))
}

func (a *App) messageError(w http.ResponseWriter, r *http.Request, status int, heading, description string) {
	data := a.newPageData(r)
	data.Title = a.translate("openblur_error_page_title")
	data.InlineStyle = "#openblur-error { color: var(--color-text); text-align: center; }"
	html := a.renderPage(data, func(v *view) {
		v.raw(`<section id="openblur-error"><div><h1 aria-label="Error explanation heading">`)
		v.esc(heading)
		v.raw(`</h1><p aria-label="Error explanation">`)
		v.esc(description)
		v.raw(`</p></div></section>`)
	})
	a.writePage(w, status, html)
}

func (a *App) genericError(w http.ResponseWriter, r *http.Request, err error) {
	name, message, contextText := errorMessage(err)
	data := a.newPageData(r)
	data.Title = a.translate("openblur_error_page_title")
	data.InlineStyle = "#openblur-error > div { background-color: var(--color-top-level-card-bg); border-radius: 0; padding: 25px; max-width: 100%; } #openblur-error { color: var(--color-text); } #openblur-error details { margin-top: 20px; } #openblur-error pre { text-wrap: wrap; } #openblur-error a { text-decoration: underline; } #error-header { font-size: 16px; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 10px; }"
	html := a.renderPage(data, func(v *view) {
		v.raw(`<section id="openblur-error"><div class="card"><div id="error-header"><h2>`)
		v.esc(a.translate("openblur_error_generic"))
		v.raw(`</h2><p>`)
		v.esc(a.translate("openblur_error_generic_description"))
		v.raw(`</p><p>`)
		v.esc(a.translate("openblur_error_generic_description_2"))
		v.raw(`</p></div>`)
		a.renderErrorDetails(v, name, message, contextText, true)
		v.raw(`</div></section>`)
	})
	a.writePage(w, http.StatusInternalServerError, html)
}

func errorMessage(err error) (string, string, string) {
	name := fmt.Sprintf("%T", err)
	message := err.Error()
	contextText := string(debug.Stack())
	return name, message, contextText
}

// handleAPIError maps Tumblr API errors to user-facing pages.
func (a *App) handleAPIError(w http.ResponseWriter, r *http.Request, err error) bool {
	var apiErr *tumblr.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.Kind {
		case tumblr.ErrLoginRequired:
			a.messageError(w, r, http.StatusForbidden, a.translate("tumblr_error_blog_login_required_error_heading"), a.translate("tumblr_error_blog_login_required_error_description"))
			return true
		case tumblr.ErrPasswordRequired:
			a.messageError(w, r, http.StatusForbidden, a.translate("tumblr_error_blog_requires_password_error_heading"), a.translate("tumblr_error_blog_login_required_error_description"))
			return true
		case tumblr.ErrRestrictedTag:
			a.messageError(w, r, http.StatusForbidden, a.translate("tumblr_error_restricted_tag_error_heading"), a.translate("tumblr_error_restricted_tag_description"))
			return true
		case tumblr.ErrBlogNotFound:
			a.messageError(w, r, http.StatusNotFound, a.translate("tumblr_error_blog_not_found_error_heading"), a.translate("tumblr_error_blog_not_found_error_description"))
			return true
		}
	}
	return false
}

func (a *App) fail(w http.ResponseWriter, r *http.Request, err error) {
	if a.handleAPIError(w, r, err) {
		return
	}
	a.genericError(w, r, err)
}

// newPageData builds page data with request-level fields populated.
func (a *App) newPageData(r *http.Request) *PageData {
	prefs := preferencesFrom(r)

	path := strings.TrimSuffix(r.URL.Path, "/rss")
	isRSS := path != r.URL.Path
	pageURL := path
	if r.URL.RawQuery != "" {
		pageURL += "?" + r.URL.RawQuery
	}

	return &PageData{
		Lang:        "en_US",
		Path:        path,
		QueryString: r.URL.RawQuery,
		ExpandPosts: prefs.ExpandPosts,
		RSS:         isRSS,
		PageURL:     pageURL,
		Updated:     time.Now().UTC(),
	}
}
