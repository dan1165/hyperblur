package web

import (
	"bytes"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dan1165/hyperblur/npf"
	"github.com/dan1165/hyperblur/tumblr"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/klauspost/compress/gzip"
	"golang.org/x/sync/singleflight"
)

// Handler builds HTTP handler for app.
func (a *App) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /assets/{path...}", a.handleAsset)

	mux.HandleFunc("GET /{$}", a.handleRoot)
	mux.HandleFunc("GET /robots.txt", a.handleRobots)
	mux.HandleFunc("GET /manifest.webmanifest", a.handleManifest)
	mux.HandleFunc("GET /sw.js", a.handleServiceWorker)
	mux.HandleFunc("GET /offline", a.handleOffline)
	mux.HandleFunc("GET /healthz", a.handleHealth)

	// Media proxying.
	mux.HandleFunc("GET /tblr/media/{cdn}/{path...}", a.handleMediaCDN)
	mux.HandleFunc("GET /tblr/a/{path...}", a.handleMediaAudio)
	mux.HandleFunc("GET /tblr/assets/{path...}", a.handleMediaAssets)
	mux.HandleFunc("GET /tblr/static/{path...}", a.handleMediaStatic)

	// at.tumblr.com link redirects.
	mux.HandleFunc("GET /at/{path...}", a.handleAtLinks)

	// API.
	mux.HandleFunc("GET /api/v1/poll/{blog}/{post_id}/{poll_id}/results", a.handleAPIPollResults)

	// Blog pages go through fallback handler: ServeMux rejects their wildcard
	// patterns next to /assets/.
	mux.HandleFunc("GET /", a.handleFallback)

	// Security headers wrap the app; the page cache sits outside so cached
	// responses carry the same headers without re-running the middleware.
	a.inner = a.middleware(mux)
	a.prewarm()
	return a.pageCache(a.inner)
}

// handleFallback dispatches explore/search/tagged/blog paths. Not ServeMux
// patterns: wildcards clash with each other and /assets/.
func (a *App) handleFallback(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.Trim(r.URL.Path, "/")
	if trimmed == "" {
		a.handleRoot(w, r)
		return
	}
	segments := strings.Split(trimmed, "/")

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
		a.notFound(w, r)
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
	if len(segments) != 2 {
		a.notFound(w, r)
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
			a.notFound(w, r)
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
			a.notFound(w, r)
			return
		}
		r.SetPathValue("query", segments[1])
		r.SetPathValue("post_filter", segments[3])
		a.handleSearchRecentFilter(w, r)
	default:
		a.notFound(w, r)
	}
}

func (a *App) routeTagged(w http.ResponseWriter, r *http.Request, segments []string) {
	if len(segments) != 2 {
		a.notFound(w, r)
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
		header := w.Header()
		header.Set("x-xss-protection", "1; mode=block")
		header.Set("x-content-type-options", "nosniff")
		header.Set("referrer-policy", "same-origin")
		header.Set("cache-control", "no-store, no-cache, must-revalidate, max-age=0")
		header.Set("pragma", "no-cache")
		header.Set("expires", "0")
		imgSrc := "'self' data:"
		mediaSrc := "'self'"
		if !ProxyMedia {
			// Direct mode: media is loaded straight from Tumblr's CDN.
			imgSrc += " https://*.tumblr.com"
			mediaSrc += " https://*.tumblr.com"
		}
		header.Set("content-security-policy", strings.Join([]string{
			"default-src 'none'",
			"script-src 'self'",
			"style-src 'self' 'unsafe-inline'",
			"img-src " + imgSrc,
			"font-src 'self' data:",
			"connect-src 'self'",
			"manifest-src 'self'",
			"worker-src 'self'",
			"media-src " + mediaSrc,
			"child-src 'self' blob:",
			"base-uri 'none'",
			"form-action 'self'",
			"frame-ancestors 'none'",
		}, "; "))
		header.Set("x-frame-options", "DENY")

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
	entry := a.assets["robots.txt"]
	if entry == nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("content-type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(entry.body)))
	_, _ = w.Write(entry.body)
}

// applyMediaMode reads HYPERBLUR_PROXY_MEDIA. Unset or truthy proxies media;
// a falsey value loads it straight from Tumblr's CDN.
func applyMediaMode() {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(proxyMediaEnvVar))) {
	case "0", "false", "no", "off":
		ProxyMedia = false
	default:
		ProxyMedia = true
	}
}

func (a *App) handleManifest(w http.ResponseWriter, r *http.Request) {
	entry := a.assets["manifest.webmanifest"]
	if entry == nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("content-type", "application/manifest+json; charset=utf-8")
	w.Header().Set("cache-control", "no-cache")
	w.Header().Set("Content-Length", strconv.Itoa(len(entry.body)))
	_, _ = w.Write(entry.body)
}

func (a *App) handleServiceWorker(w http.ResponseWriter, r *http.Request) {
	entry := a.assets["sw.js"]
	if entry == nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("content-type", "text/javascript; charset=utf-8")
	w.Header().Set("cache-control", "no-cache")
	w.Header().Set("service-worker-allowed", "/")
	w.Header().Set("Content-Length", strconv.Itoa(len(entry.body)))
	_, _ = w.Write(entry.body)
}

func (a *App) handleOffline(w http.ResponseWriter, r *http.Request) {
	data := a.newPageData(r)
	data.Title = "Offline"
	a.writePage(w, r, http.StatusOK, a.renderPage(data, func(v *view) {
		v.raw(`<section id="hyperblur-error"><div><h1>`)
		v.esc("You are offline")
		v.raw(`</h1><p>`)
		v.esc("hyperblur needs a connection to reach Tumblr. Reconnect and try again.")
		v.raw(`</p></div></section>`)
	}))
}

// ---------------------------------------------------------------------------
// Error rendering
// ---------------------------------------------------------------------------

func (a *App) writePage(w http.ResponseWriter, r *http.Request, status int, html string) {
	header := w.Header()
	header.Set("Content-Type", "text/html; charset=utf-8")

	body := []byte(html)
	// Cacheable pages are buffered raw by the page cache, which negotiates the
	// client encoding from the precomputed gzip variant.
	if a.caching && cacheablePath(r.URL.Path) {
		header.Set("Content-Length", strconv.Itoa(len(body)))
		w.WriteHeader(status)
		_, _ = w.Write(body)
		return
	}

	if acceptsGzip(r) {
		body = gzipCompress(body)
		header.Set("Content-Encoding", "gzip")
		header.Add("Vary", "Accept-Encoding")
	}

	header.Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func (a *App) messageError(w http.ResponseWriter, r *http.Request, status int, heading, description string) {
	// No inline style: base.css already carries the error page chrome.
	a.renderErrorPage(w, r, status, "", func(v *view) {
		v.raw(`<section id="hyperblur-error"><div><h1 aria-label="Error explanation heading">`)
		v.esc(heading)
		v.raw(`</h1><p aria-label="Error explanation">`)
		v.esc(description)
		v.raw(`</p></div></section>`)
	})
}

// renderErrorPage renders shared error-page chrome around center.
func (a *App) renderErrorPage(w http.ResponseWriter, r *http.Request, status int, style string, center func(v *view)) {
	data := a.newPageData(r)
	data.Title = "Error"
	data.InlineStyle = style
	a.writePage(w, r, status, a.renderPage(data, center))
}

func (a *App) genericError(w http.ResponseWriter, r *http.Request, err error) {
	a.Logger.Printf("error handling %s %s: %v", r.Method, r.URL.RequestURI(), err)

	name, message, contextText := errorMessage(err)
	contextText = fmt.Sprintf("Request: %s %s\n\n%s", r.Method, r.URL.RequestURI(), contextText)

	a.renderErrorPage(w, r, http.StatusInternalServerError,
		"#hyperblur-error > div { padding: var(--s-5); border: 1px solid var(--rule); background: var(--surface-2); text-align: left; }"+
			" #error-header { display: flex; flex-direction: column; align-items: center; gap: var(--s-2); text-align: center; }"+
			" #error-header h2 { margin: 0; font-size: var(--text-lg); }", func(v *view) {
			v.raw(`<section id="hyperblur-error"><div class="card"><div id="error-header"><h2>`)
			v.esc("An unknown exception has occured!")
			v.raw(`</h2><p>`)
			v.esc("It looks like you have found a bug in hyperblur.")
			v.raw(`</p><p><a href="https://github.com/dan1165/hyperblur/issues">`)
			v.esc("Please report it here at GitHub")
			v.raw(`</a></p></div>`)
			a.renderErrorDetails(v, name, message, contextText, true)
			v.raw(`</div></section>`)
		})
}

func errorMessage(err error) (string, string, string) {
	name := fmt.Sprintf("%T", err)
	message := err.Error()
	contextText := string(debug.Stack())
	return name, message, contextText
}

// handleAPIError maps Tumblr API errors to user pages.
func (a *App) handleAPIError(w http.ResponseWriter, r *http.Request, err error) bool {
	var apiErr *tumblr.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	a.Logger.Printf("tumblr error on %s %s: kind=%d code=%d internal=%d: %s", r.Method, r.URL.RequestURI(), apiErr.Kind, apiErr.HTTPCode, apiErr.InternalCode, apiErr.Message)

	switch apiErr.Kind {
	case tumblr.ErrLoginRequired:
		a.messageError(w, r, http.StatusForbidden, "This blog requires an account to view", "Try finding reblogs instead!")
		return true
	case tumblr.ErrPasswordRequired:
		a.messageError(w, r, http.StatusForbidden, "This blog requires a password to access", "Try finding reblogs instead!")
		return true
	case tumblr.ErrRestrictedTag:
		a.messageError(w, r, http.StatusForbidden, "This tag has been restricted on Tumblr", "Try performing a search instead")
		return true
	case tumblr.ErrBlogNotFound:
		a.messageError(w, r, http.StatusNotFound, "Unable to find the requested blog", "The blog may have been deleted or just never existed in the first place")
		return true
	}
	return false
}

func (a *App) fail(w http.ResponseWriter, r *http.Request, err error) {
	if a.handleAPIError(w, r, err) {
		return
	}
	a.genericError(w, r, err)
}

// notFound renders the styled 404 for HTML routes. Media and asset routes keep
// net/http's plain text: they are not pages.
func (a *App) notFound(w http.ResponseWriter, r *http.Request) {
	a.messageError(w, r, http.StatusNotFound,
		"Nothing here",
		"That page does not exist. Check the address, or search for something else.")
}

// newPageData builds page data with request fields.
func (a *App) newPageData(r *http.Request) *PageData {
	return &PageData{
		Ctx:  r.Context(),
		Path: r.URL.Path,
		// The first few images load eagerly (avatar + first post media);
		// everything below the fold waits for the scroll.
		EagerImages: 3,
	}
}

// App holds shared server state.
type App struct {
	API   *tumblr.API
	Media *http.Client
	// At resolves at.tumblr.com links. It must not follow redirects, so the
	// 301 can be inspected.
	At     *http.Client
	Logger *log.Logger
	assets map[string]*assetEntry

	// Caching + background refresh.
	inner      http.Handler
	pages      *expirable.LRU[string, *pageResponse]
	polls      *expirable.LRU[string, []byte]
	sf         singleflight.Group
	pollsSF    singleflight.Group
	refreshSF  singleflight.Group
	caching    bool
	cacheTTL   time.Duration
	cacheStale time.Duration

	// Precomputed inline stylesheets.
	css     string
	cssBlog string
}

// NewApp builds app and Tumblr API client.
func NewApp(logger *log.Logger) *App {
	applyMediaMode()
	transport := newTransport()
	assets := loadAssets()

	app := &App{
		API:        tumblr.NewAPI(transport, 10*time.Second),
		Media:      &http.Client{Transport: transport, Timeout: 30 * time.Second},
		At:         &http.Client{Transport: transport, Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
		Logger:     logger,
		assets:     assets,
		caching:    envBool("HYPERBLUR_CACHE", true),
		cacheTTL:   envSeconds("HYPERBLUR_CACHE_TTL", 60),
		cacheStale: envSeconds("HYPERBLUR_CACHE_STALE", 600),
		css:        inlineCSS(assets, ""),
		cssBlog:    inlineCSS(assets, "css/blog.css"),
	}
	if app.caching {
		app.pages = expirable.NewLRU[string, *pageResponse](512, nil, app.cacheTTL+app.cacheStale)
		app.polls = expirable.NewLRU[string, []byte](512, nil, 18*time.Second)
	}
	if app.API.UsingBundledToken {
		app.Logger.Print("warning: using the bundled Tumblr API token; set HYPERBLUR_TUMBLR_API_TOKEN to use your own")
	}
	return app
}

// inlineCSS concatenates the embedded stylesheets once, so a page render never
// builds them per request. Rules are minified on the way in: the sheets are
// commented for maintenance, and the browser does not need the comments.
func inlineCSS(assets map[string]*assetEntry, extra string) string {
	var b strings.Builder
	for _, key := range []string{"css/base.css", "css/post.css"} {
		if e := assets[key]; e != nil {
			b.WriteString(minifyCSS(e.body))
		}
	}
	if extra != "" {
		if e := assets[extra]; e != nil {
			b.WriteString(minifyCSS(e.body))
		}
	}
	return b.String()
}

// cssComment matches /* ... */, which the stylesheets never nest and never
// place inside a string.
var cssComment = regexp.MustCompile(`/\*(?:[^*]|\*+[^*/])*\*+/`)

// minifyCSS drops comments and per-line indentation. Both are safe: CSS strings
// cannot contain a literal newline, and comments never carry meaning.
func minifyCSS(css []byte) string {
	stripped := cssComment.ReplaceAll(css, nil)

	var b strings.Builder
	b.Grow(len(stripped))
	atLineStart := true
	for _, c := range stripped {
		if c == '\n' {
			atLineStart = true
			b.WriteByte(c)
			continue
		}
		if atLineStart && (c == ' ' || c == '\t') {
			continue
		}
		atLineStart = false
		b.WriteByte(c)
	}
	return b.String()
}

// pageCSS returns the precomputed stylesheet for a page.
func (a *App) pageCSS(blog bool) string {
	if blog {
		return a.cssBlog
	}
	return a.css
}

// prewarm fetches the common entry points into the cache in the background, so
// the first visitor already hits a warm cache.
func (a *App) prewarm() {
	if !a.caching {
		return
	}
	for _, path := range []string{"/explore/trending", "/explore/today"} {
		req := &http.Request{
			Method: http.MethodGet,
			URL:    &url.URL{Path: path},
			Header: make(http.Header),
		}
		req.Header.Set("Accept-Encoding", "gzip")
		a.refresh(cacheKey(req), req)
	}
}

func envBool(name string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "0", "false", "no", "off":
		return false
	case "1", "true", "yes", "on":
		return true
	}
	return fallback
}

func envSeconds(name string, fallback int) time.Duration {
	if v, err := strconv.Atoi(strings.TrimSpace(os.Getenv(name))); err == nil && v > 0 {
		return time.Duration(v) * time.Second
	}
	return time.Duration(fallback) * time.Second
}

// handleHealth is a tiny health/observability endpoint.
func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	pages, polls := 0, 0
	if a.pages != nil {
		pages = a.pages.Len()
	}
	if a.polls != nil {
		polls = a.polls.Len()
	}
	w.Header().Set("content-type", "text/plain; charset=utf-8")
	w.Header().Set("x-hyperblur-version", Version)
	fmt.Fprintf(w, "ok\nversion: %s\ncache: %t\npage entries: %d\npoll entries: %d\n", Version, a.caching, pages, polls)
}

// newTransport returns an HTTP transport tuned for many concurrent Tumblr
// requests: a large idle pool, larger socket buffers, and HTTP/2 with idle
// pings so connections stay warm and dead peers are detected quickly. HTTP/2
// settings come from the standard library (no golang.org/x/net dependency).
func newTransport() *http.Transport {
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetHTTP2(true)

	return &http.Transport{
		Proxy:                  http.ProxyFromEnvironment,
		DialContext:            (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		Protocols:              protocols,
		MaxIdleConns:           512,
		MaxIdleConnsPerHost:    100,
		IdleConnTimeout:        90 * time.Second,
		TLSHandshakeTimeout:    10 * time.Second,
		ExpectContinueTimeout:  time.Second,
		ResponseHeaderTimeout:  20 * time.Second,
		MaxResponseHeaderBytes: 64 << 10,
		WriteBufferSize:        64 << 10,
		ReadBufferSize:         64 << 10,
		HTTP2: &http.HTTP2Config{
			// Ping an idle connection, and drop it if the peer never answers.
			SendPingTimeout: 30 * time.Second,
			PingTimeout:     15 * time.Second,
		},
	}
}

// formatNPF renders post content, returns optional render error.
func (a *App) formatNPF(ctx context.Context, content, layout []any, blogName, postID string, fetchPolls bool, eagerBudget *int) (*renderError, string) {
	params := params{
		BlogName:    blogName,
		PostID:      postID,
		Ctx:         ctx,
		EagerBudget: eagerBudget,
	}
	if fetchPolls && blogName != "" && postID != "" {
		params.PollCallback = a.pollCallback(ctx, blogName, postID)
	}
	return formatNPF(content, layout, params)
}

func (a *App) pollCallback(ctx context.Context, blog, postID string) npf.PollCallback {
	if ctx == nil {
		ctx = context.Background()
	}
	return func(pollID string, _ int64) (map[string]npf.PollResult, error) {
		result, err := a.API.PollResults(ctx, blog, postID, pollID)
		if err != nil {
			return nil, err
		}
		response, _ := result["response"].(map[string]any)
		results := map[string]npf.PollResult{}
		if rawResults, ok := response["results"].(map[string]any); ok {
			for id, count := range rawResults {
				results[id] = npf.PollResult{VoteCount: int(toFloat(count))}
			}
		}
		return results, nil
	}
}

func toFloat(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	}
	return 0
}

// assetEntry is precomputed embedded file, gzipped when compressible.
type assetEntry struct {
	body  []byte
	gz    []byte
	etag  string
	ctype string
}

//go:embed assets
var assetsFS embed.FS

// loadAssets reads every embedded asset once at startup.
func loadAssets() map[string]*assetEntry {
	out := map[string]*assetEntry{}
	_ = fs.WalkDir(assetsFS, "assets", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		body, err := fs.ReadFile(assetsFS, p)
		if err != nil {
			return nil
		}
		name := strings.TrimPrefix(p, "assets/")

		sum := sha256.Sum256(body)
		entry := &assetEntry{
			body:  body,
			etag:  `"` + hex.EncodeToString(sum[:16]) + `"`,
			ctype: contentType(name),
		}
		if compressible(entry.ctype) && len(body) > 512 {
			var buf bytes.Buffer
			writer, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
			_, _ = writer.Write(body)
			_ = writer.Close()
			entry.gz = buf.Bytes()
		}
		out[name] = entry
		return nil
	})
	return out
}

func contentType(p string) string {
	if ct := mime.TypeByExtension(path.Ext(p)); ct != "" {
		return ct
	}
	return "application/octet-stream"
}

func compressible(ct string) bool {
	return strings.HasPrefix(ct, "text/") ||
		strings.Contains(ct, "javascript") ||
		strings.Contains(ct, "json") ||
		strings.Contains(ct, "svg")
}

// gzipLevel is level 6 rather than BestSpeed: compression happens once per
// cache fill and the result is served many times, and on a real 535 KB page
// level 6 costs ~0.4 ms more while shipping ~10% fewer bytes (98 KB vs 109 KB).
const gzipLevel = 6

// gzipWriters recycles gzip writers across requests. Pooled writers retain no
// measurable memory between uses, so a higher level is free at rest.
var gzipWriters = sync.Pool{
	New: func() any {
		writer, _ := gzip.NewWriterLevel(io.Discard, gzipLevel)
		return writer
	},
}

// gzipCompress gzips data with a pooled writer.
func gzipCompress(data []byte) []byte {
	var buf bytes.Buffer
	buf.Grow(len(data)/2 + 64)
	writer := gzipWriters.Get().(*gzip.Writer)
	writer.Reset(&buf)
	_, _ = writer.Write(data)
	_ = writer.Close()
	gzipWriters.Put(writer)
	return buf.Bytes()
}

// copyBuffers recycles buffers used when streaming proxied media.
var copyBuffers = sync.Pool{
	New: func() any {
		buf := make([]byte, 32<<10)
		return &buf
	},
}

// handleAsset serves embedded asset with long cache, ETag revalidation, gzip.
func (a *App) handleAsset(w http.ResponseWriter, r *http.Request) {
	entry := a.assets[r.PathValue("path")]
	if entry == nil {
		http.NotFound(w, r)
		return
	}

	header := w.Header()
	header.Set("ETag", entry.etag)
	// No "immutable": asset URLs are fixed across builds, so a revalidation
	// after max-age is what lets a deploy replace post.js.
	header.Set("Cache-Control", "public, max-age=604800")
	header.Del("pragma")
	header.Del("expires")

	if r.Header.Get("If-None-Match") == entry.etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	header.Set("Content-Type", entry.ctype)
	body := entry.body
	if entry.gz != nil && strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		header.Set("Content-Encoding", "gzip")
		header.Add("Vary", "Accept-Encoding")
		body = entry.gz
	}
	header.Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

var mediaRequestHeaders = [][2]string{
	{"user-agent", tumblr.DefaultUserAgent},
	{"accept", "image/avif,image/webp,image/png,image/svg+xml,image/*;q=0.8,*/*;q=0.5"},
	{"accept-language", "en-US,en;q=0.5"},
	{"connection", "keep-alive"},
	{"te", "trailers"},
	{"referer", "https://www.tumblr.com/"},
}

var forwardedRequestHeaders = []string{"range", "if-range", "if-none-match", "if-modified-since"}

// Headers hyperblur never forwards from upstream media. Origin-scoped security
// headers and cookies have to go: they would otherwise apply to hyperblur's own
// origin, not Tumblr's.
var blacklistResponseHeaders = map[string]bool{
	"access-control-allow-origin":         true,
	"alt-svc":                             true,
	"server":                              true,
	"set-cookie":                          true,
	"strict-transport-security":           true,
	"content-security-policy-report-only": true,
	"content-disposition":                 true,
	"report-to":                           true,
	"nel":                                 true,
	"origin-trial":                        true,
	"permissions-policy":                  true,
	"cross-origin-opener-policy":          true,
	"cross-origin-embedder-policy":        true,
	"cross-origin-resource-policy":        true,
	// Set by our middleware; never duplicate.
	"cache-control":           true,
	"pragma":                  true,
	"expires":                 true,
	"x-xss-protection":        true,
	"x-content-type-options":  true,
	"referrer-policy":         true,
	"content-security-policy": true,
	"x-frame-options":         true,
}

const (
	videoAccept       = "video/webm,video/ogg,video/*;q=0.9, application/ogg;q=0.7,audio/*;q=0.6,*/*;q=0.5"
	audioAccept       = "audio/webm,audio/ogg,audio/wav,audio/*;q=0.9,application/ogg;q=0.7,video/*;q=0.6,*/*;q=0.5"
	maxFilenameLength = 150
)

func (a *App) handleMediaCDN(w http.ResponseWriter, r *http.Request) {
	cdn := r.PathValue("cdn")
	if !validCDNName(cdn) {
		http.NotFound(w, r)
		return
	}
	extra := map[string]string(nil)
	if cdn == "ve" || cdn == "va" {
		extra = map[string]string{"accept": videoAccept}
	}
	a.serveMedia(w, r, "https://"+cdn+".media.tumblr.com", extra)
}

func (a *App) handleMediaAudio(w http.ResponseWriter, r *http.Request) {
	a.serveMedia(w, r, "https://a.tumblr.com", map[string]string{"accept": audioAccept})
}

func (a *App) handleMediaAssets(w http.ResponseWriter, r *http.Request) {
	a.serveMedia(w, r, "https://assets.tumblr.com", nil)
}

func (a *App) handleMediaStatic(w http.ResponseWriter, r *http.Request) {
	a.serveMedia(w, r, "https://static.tumblr.com", nil)
}

// serveMedia proxies media through hyperblur, so the browser never contacts
// Tumblr's CDN; a forced download adds a content-disposition filename.
func (a *App) serveMedia(w http.ResponseWriter, r *http.Request, origin string, extraHeaders map[string]string) {
	path := r.PathValue("path")
	if !validMediaPath(path) {
		http.NotFound(w, r)
		return
	}

	target := origin + "/" + quotePath(path)
	if query := mediaUpstreamQuery(r.URL.RawQuery); query != "" {
		target += "?" + query
	}
	disposition := ""
	if downloadRequested(r) {
		disposition = buildContentDisposition(path)
	}
	a.streamMedia(w, r, target, extraHeaders, disposition)
}

// mediaUpstreamQuery drops hyperblur's own download flag and passes the rest of
// the query through byte-for-byte, so signed CDN URLs stay valid.
func mediaUpstreamQuery(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}
	kept := make([]string, 0, 4)
	for _, pair := range strings.Split(rawQuery, "&") {
		if key, _, _ := strings.Cut(pair, "="); key == "download" {
			continue
		}
		kept = append(kept, pair)
	}
	return strings.Join(kept, "&")
}

func (a *App) handleAtLinks(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("path")
	if !validMediaPath(path) {
		http.NotFound(w, r)
		return
	}
	request, err := http.NewRequestWithContext(r.Context(), http.MethodHead, "https://at.tumblr.com/"+quotePath(path), nil)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	request.Header.Set("user-agent", tumblr.DefaultUserAgent)

	response, err := a.At.Do(request)
	if err != nil {
		a.messageError(w, r, http.StatusBadGateway, "Error: could not reach Tumblr to resolve this link", "")
		return
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusMovedPermanently {
		if location, err := url.Parse(response.Header.Get("location")); err == nil && strings.HasPrefix(location.Path, "/") {
			http.Redirect(w, r, location.Path, http.StatusFound)
			return
		}
	}
	a.messageError(w, r, http.StatusBadGateway, "Error: Tumblr did not redirect this link to a hyperblur page", "")
}

func (a *App) streamMedia(w http.ResponseWriter, r *http.Request, origin string, extraHeaders map[string]string, downloadFilename string) {
	request, err := http.NewRequestWithContext(r.Context(), http.MethodGet, origin, nil)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	for _, header := range mediaRequestHeaders {
		request.Header.Set(header[0], header[1])
	}
	for key, value := range extraHeaders {
		request.Header.Set(key, value)
	}
	for _, header := range forwardedRequestHeaders {
		if value := r.Header.Get(header); value != "" {
			request.Header.Set(header, value)
		}
	}

	response, err := a.Media.Do(request)
	if err != nil {
		a.Logger.Printf("failed to fetch media %s: %v", origin, err)
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	defer response.Body.Close()

	// Cache proxied media in the browser: honour upstream caching, else a day.
	upstreamCacheControl := response.Header.Get("cache-control")

	for key, values := range response.Header {
		if blacklistResponseHeaders[strings.ToLower(key)] {
			continue
		}
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	if upstreamCacheControl == "" {
		upstreamCacheControl = "public, max-age=86400"
	}
	w.Header().Set("cache-control", upstreamCacheControl)
	w.Header().Del("pragma")
	w.Header().Del("expires")
	if downloadFilename != "" {
		w.Header().Set("content-disposition", downloadFilename)
	}

	w.WriteHeader(response.StatusCode)
	buf := copyBuffers.Get().(*[]byte)
	if _, err := io.CopyBuffer(w, response.Body, *buf); err != nil {
		a.Logger.Printf("media stream interrupted for %s: %v", origin, err)
	}
	copyBuffers.Put(buf)
}

func validCDNName(cdn string) bool {
	if cdn == "" || len(cdn) > 63 {
		return false
	}
	for i := 0; i < len(cdn); i++ {
		c := cdn[i]
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-') {
			return false
		}
	}
	return true
}

// validMediaPath rejects NUL bytes and ".." segments. Checked without splitting
// the path: this runs for every proxied media request.
func validMediaPath(path string) bool {
	decoded, err := url.PathUnescape(path)
	if err != nil {
		decoded = path
	}
	if strings.ContainsRune(decoded, 0) || decoded == ".." {
		return false
	}
	return !strings.Contains(decoded, "/../") &&
		!strings.HasPrefix(decoded, "../") &&
		!strings.HasSuffix(decoded, "/..")
}

func downloadRequested(r *http.Request) bool {
	return truthyQuery(r.URL.Query().Get("download"))
}

// quotePath percent-encodes path, keeps "/" separators.
func quotePath(path string) string {
	return percentEncode(path, "/")
}

func buildContentDisposition(path string) string {
	decoded, err := url.PathUnescape(path)
	if err != nil {
		decoded = path
	}
	filename := decoded
	if idx := strings.LastIndexByte(decoded, '/'); idx >= 0 {
		filename = decoded[idx+1:]
	}

	var cleaned strings.Builder
	for _, ch := range filename {
		if ch == '"' || ch == '\\' || ch == '\r' || ch == '\n' {
			continue
		}
		cleaned.WriteRune(ch)
	}
	filename = cleaned.String()
	if len(filename) > maxFilenameLength {
		filename = filename[:maxFilenameLength]
	}
	if filename == "" {
		filename = "download"
	}

	var ascii strings.Builder
	for _, ch := range filename {
		if ch >= 32 && ch < 127 {
			ascii.WriteRune(ch)
		}
	}
	asciiFilename := ascii.String()
	if asciiFilename == "" {
		asciiFilename = "download"
	}

	return `attachment; filename="` + asciiFilename + `"; filename*=UTF-8''` + quotePath(filename)
}
