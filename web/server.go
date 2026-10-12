package web

import (
	"bytes"
	"compress/gzip"
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
	"path"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/dan1165/hyperblur/npf"
	"github.com/dan1165/hyperblur/tumblr"
)

// Handler builds the HTTP handler for the application.
func (a *App) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /assets/{path...}", a.handleAsset)

	mux.HandleFunc("GET /{$}", a.handleRoot)
	mux.HandleFunc("GET /robots.txt", a.handleRobots)

	// Media proxying.
	mux.HandleFunc("GET /tblr/media/{cdn}/{path...}", a.handleMediaCDN)
	mux.HandleFunc("GET /tblr/a/{path...}", a.handleMediaAudio)
	mux.HandleFunc("GET /tblr/assets/{path...}", a.handleMediaAssets)
	mux.HandleFunc("GET /tblr/static/{path...}", a.handleMediaStatic)

	// at.tumblr.com link redirects.
	mux.HandleFunc("GET /at/{path...}", a.handleAtLinks)

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
	if len(segments) != 2 {
		http.NotFound(w, r)
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
	if len(segments) != 2 {
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
	entry := a.assets["robots.txt"]
	if entry == nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("content-type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(entry.body)))
	_, _ = w.Write(entry.body)
}

// ---------------------------------------------------------------------------
// Error rendering
// ---------------------------------------------------------------------------

func (a *App) writePage(w http.ResponseWriter, r *http.Request, status int, html string) {
	header := w.Header()
	header.Set("Content-Type", "text/html; charset=utf-8")

	body := []byte(html)
	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		var buf bytes.Buffer
		writer, _ := gzip.NewWriterLevel(&buf, gzip.BestSpeed)
		_, _ = writer.Write(body)
		_ = writer.Close()
		header.Set("Content-Encoding", "gzip")
		header.Add("Vary", "Accept-Encoding")
		body = buf.Bytes()
	}

	header.Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func (a *App) messageError(w http.ResponseWriter, r *http.Request, status int, heading, description string) {
	a.renderErrorPage(w, r, status, "#hyperblur-error { color: var(--color-text); text-align: center; }", func(v *view) {
		v.raw(`<section id="hyperblur-error"><div><h1 aria-label="Error explanation heading">`)
		v.esc(heading)
		v.raw(`</h1><p aria-label="Error explanation">`)
		v.esc(description)
		v.raw(`</p></div></section>`)
	})
}

// renderErrorPage renders the shared error-page chrome around center.
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

	a.renderErrorPage(w, r, http.StatusInternalServerError, "#hyperblur-error > div { background-color: var(--color-top-level-card-bg); border-radius: 0; padding: 25px; max-width: 100%; } #hyperblur-error { color: var(--color-text); } #hyperblur-error details { margin-top: 20px; } #hyperblur-error pre { text-wrap: wrap; } #hyperblur-error a { text-decoration: underline; } #error-header { font-size: 16px; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 10px; }", func(v *view) {
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

// handleAPIError maps Tumblr API errors to user-facing pages.
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

// newPageData builds page data with request-level fields populated.
func (a *App) newPageData(r *http.Request) *PageData {
	return &PageData{Path: r.URL.Path}
}

// App holds the shared server state.
type App struct {
	API    *tumblr.API
	Media  *http.Client
	Logger *log.Logger
	assets map[string]*assetEntry
}

// NewApp builds the application, creating the Tumblr API client.
func NewApp(logger *log.Logger) *App {
	transport := newTransport()
	return &App{
		API:    tumblr.NewAPI(transport, 10*time.Second),
		Media:  &http.Client{Transport: transport, Timeout: 30 * time.Second},
		Logger: logger,
		assets: loadAssets(),
	}
}

// newTransport returns an HTTP transport with a larger idle pool than the
// default (which keeps only 2 idle connections per host) so that concurrent
// requests to Tumblr reuse TLS connections instead of re-handshaking.
func newTransport() *http.Transport {
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          200,
		MaxIdleConnsPerHost:   100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: time.Second,
	}
}

// FormatNPF renders post content and returns an optional render error.
func (a *App) FormatNPF(content, layout []any, blogName, postID string, fetchPolls bool) (*RenderError, string) {
	params := Params{
		BlogName: blogName,
		PostID:   postID,
	}
	if fetchPolls && blogName != "" && postID != "" {
		params.PollCallback = a.pollCallback(blogName, postID)
	}
	return FormatNPF(content, layout, params)
}

func (a *App) pollCallback(blog, postID string) npf.PollCallback {
	return func(pollID string, _ int64) (map[string]npf.PollResult, error) {
		result, err := a.API.PollResults(blog, postID, pollID)
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

// assetEntry is a precomputed embedded file, gzipped when it is compressible.
type assetEntry struct {
	body  []byte
	gz    []byte
	etag  string
	ctype string
}

//go:embed assets
var assetsFS embed.FS

// loadAssets reads every embedded asset once, at startup.
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

// handleAsset serves an embedded asset with long-lived caching, ETag
// revalidation and gzip.
func (a *App) handleAsset(w http.ResponseWriter, r *http.Request) {
	entry := a.assets[r.PathValue("path")]
	if entry == nil {
		http.NotFound(w, r)
		return
	}

	header := w.Header()
	header.Set("ETag", entry.etag)
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

var mediaRequestHeaders = map[string]string{
	"user-agent":      tumblr.DefaultUserAgent,
	"accept":          "image/avif,image/webp,image/png,image/svg+xml,image/*;q=0.8,*/*;q=0.5",
	"accept-language": "en-US,en;q=0.5",
	"connection":      "keep-alive",
	"te":              "trailers",
	"referer":         "https://www.tumblr.com/",
}

var forwardedRequestHeaders = []string{"range", "if-range", "if-none-match", "if-modified-since"}

// Headers hyperblur never forwards from upstream media responses.
var blacklistResponseHeaders = map[string]bool{
	"access-control-allow-origin": true,
	"alt-svc":                     true,
	"server":                      true,
	// These are set by our own middleware and must not be duplicated.
	"cache-control":           true,
	"pragma":                  true,
	"expires":                 true,
	"x-xss-protection":        true,
	"x-content-type-options":  true,
	"referrer-policy":         true,
	"content-security-policy": true,
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

// serveMedia redirects a media request to its origin, or proxies it when a
// forced download is requested.
func (a *App) serveMedia(w http.ResponseWriter, r *http.Request, origin string, extraHeaders map[string]string) {
	path := r.PathValue("path")
	if !validMediaPath(path) {
		http.NotFound(w, r)
		return
	}

	target := origin + "/" + quotePath(path)
	if !downloadRequested(r) {
		http.Redirect(w, r, target, http.StatusFound)
		return
	}
	a.streamMedia(w, r, target, extraHeaders, buildContentDisposition(path))
}

func (a *App) handleAtLinks(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("path")
	request, err := http.NewRequestWithContext(r.Context(), http.MethodHead, "https://at.tumblr.com/"+path, nil)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	request.Header.Set("user-agent", tumblr.DefaultUserAgent)

	response, err := a.Media.Do(request)
	if err != nil {
		a.messageError(w, r, http.StatusBadGateway, "Error: Tumblr HTTP 301 redirect points to foreign URL", "")
		return
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusMovedPermanently {
		if location, err := url.Parse(response.Header.Get("location")); err == nil && strings.HasPrefix(location.Path, "/") {
			http.Redirect(w, r, location.Path, http.StatusFound)
			return
		}
	}
	a.messageError(w, r, http.StatusBadGateway, "Error: Tumblr HTTP 301 redirect points to foreign URL", "")
}

func (a *App) streamMedia(w http.ResponseWriter, r *http.Request, origin string, extraHeaders map[string]string, downloadFilename string) {
	request, err := http.NewRequestWithContext(r.Context(), http.MethodGet, origin, nil)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	for key, value := range mediaRequestHeaders {
		request.Header.Set(key, value)
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

	for key, values := range response.Header {
		if blacklistResponseHeaders[strings.ToLower(key)] {
			continue
		}
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	if downloadFilename != "" {
		w.Header().Set("content-disposition", downloadFilename)
	}

	w.WriteHeader(response.StatusCode)
	if _, err := io.Copy(w, response.Body); err != nil {
		a.Logger.Printf("media stream interrupted for %s: %v", origin, err)
	}
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

func validMediaPath(path string) bool {
	decoded, err := url.PathUnescape(path)
	if err != nil {
		decoded = path
	}
	if strings.ContainsRune(decoded, 0) {
		return false
	}
	for _, segment := range strings.Split(decoded, "/") {
		if segment == ".." {
			return false
		}
	}
	return true
}

func downloadRequested(r *http.Request) bool {
	return truthyQuery(r.URL.Query().Get("download"))
}

// quotePath percent-encodes a path, keeping "/" separators.
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
