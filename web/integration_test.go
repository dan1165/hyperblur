package web

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dan1165/hyperblur/tumblr"
	"github.com/hashicorp/golang-lru/v2/expirable"
)

// fakeUpstream answers hyperblur's Tumblr requests with canned data and counts
// what was asked for, so handlers can be tested end to end without the network.
type fakeUpstream struct {
	mu     sync.Mutex
	calls  map[string]int
	status map[string]int
	last   http.Header
}

func newFakeUpstream() *fakeUpstream {
	return &fakeUpstream{calls: map[string]int{}, status: map[string]int{}}
}

// lastRequest returns the headers of the most recent upstream request.
func (f *fakeUpstream) lastRequest() http.Header {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.last
}

// countContaining reports how many upstream requests had substr in their
// "host/path?query" key.
func (f *fakeUpstream) countContaining(substr string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	total := 0
	for key, n := range f.calls {
		if strings.Contains(key, substr) {
			total += n
		}
	}
	return total
}

func (f *fakeUpstream) setStatus(key string, status int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.status[key] = status
}

const exploreJSON = `{"response":{"timeline":{"elements":[
	{"objectType":"post","id":"42","blog":{"name":"staff","title":"Staff","active":true},
	 "content":[{"type":"text","subtype":"heading1","text":"Hello"},{"type":"text","text":"body text"}],
	 "noteCount":3,"tags":["one","two"],"timestamp":1700000000,"slug":"hello"}],
	"links":{"next":{"queryParams":{"cursor":"cursor-2"}}}}}}`

const pollJSON = `{"response":{"results":{"a1":7,"a2":3},"totalVotes":10}}`

var mediaBody = []byte("fake-image-bytes")

func (f *fakeUpstream) RoundTrip(req *http.Request) (*http.Response, error) {
	key := req.URL.Host + req.URL.Path + "?" + req.URL.RawQuery
	f.mu.Lock()
	f.calls[key]++
	f.last = req.Header.Clone()
	status, hasStatus := f.status[key]
	f.mu.Unlock()
	if !hasStatus {
		status = http.StatusOK
	}

	header := http.Header{}
	if req.URL.Host == "www.tumblr.com" {
		header.Set("Content-Type", "application/json")
		switch {
		case strings.Contains(req.URL.Path, "/api/v2/polls/"):
			return fakeResponse(req, status, header, []byte(pollJSON)), nil
		case strings.Contains(req.URL.Path, "/timeline"),
			strings.Contains(req.URL.Path, "/explore/"),
			strings.Contains(req.URL.Path, "/search"):
			return fakeResponse(req, status, header, []byte(exploreJSON)), nil
		default:
			// Blog endpoints, so 404 pages and blog routes can be exercised.
			return fakeResponse(req, http.StatusNotFound, header, []byte(`{"meta":{"status":404,"msg":"Not Found"}}`)), nil
		}
	}

	// Media CDN: include an ETag and headers that must not reach the client.
	header.Set("Content-Type", "image/jpeg")
	header.Set("ETag", `"upstream-etag"`)
	header.Set("Cache-Control", "max-age=315360000")
	header.Set("Set-Cookie", "cdn=1")
	header.Set("Strict-Transport-Security", "max-age=99999")
	return fakeResponse(req, status, header, mediaBody), nil
}

func fakeResponse(req *http.Request, status int, header http.Header, body []byte) *http.Response {
	return &http.Response{
		StatusCode:    status,
		Status:        http.StatusText(status),
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        header,
		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)),
		Request:       req,
	}
}

// newTestApp wires an App against the fake upstream.
func newTestApp(t *testing.T, upstream *fakeUpstream) *App {
	t.Helper()
	assets := loadAssets()
	app := &App{
		API:        tumblr.NewAPI(upstream, 5*time.Second),
		Media:      &http.Client{Transport: upstream, Timeout: 5 * time.Second},
		Logger:     log.New(io.Discard, "", 0),
		assets:     assets,
		caching:    true,
		cacheTTL:   time.Minute,
		cacheStale: 5 * time.Minute,
		css:        inlineCSS(assets, ""),
		cssBlog:    inlineCSS(assets, "css/blog.css"),
	}
	app.pages = expirable.NewLRU[string, *pageResponse](32, nil, app.cacheTTL+app.cacheStale)
	app.polls = expirable.NewLRU[string, []byte](32, nil, 18*time.Second)
	return app
}

// newTestServer starts the full handler stack against the fake upstream.
func newTestServer(t *testing.T) (*httptest.Server, *fakeUpstream) {
	t.Helper()
	old := ProxyMedia
	ProxyMedia = true
	t.Cleanup(func() { ProxyMedia = old })

	upstream := newFakeUpstream()
	server := httptest.NewServer(newTestApp(t, upstream).Handler())
	t.Cleanup(server.Close)
	return server, upstream
}

// noCompressionClient does not transparently gunzip, so Content-Encoding can be
// asserted.
func noCompressionClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{DisableCompression: true},
		Timeout:   10 * time.Second,
	}
}

func get(t *testing.T, client *http.Client, url string, headers map[string]string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func bodyOf(t *testing.T, resp *http.Response) string {
	t.Helper()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(body)
}

func TestPageCacheServesAndReuses(t *testing.T) {
	server, upstream := newTestServer(t)
	// The fake API has no blog routes: /tagged is a distinct path from the
	// prewarmed explore pages, so call counts stay deterministic.
	url := server.URL + "/tagged/cats"

	first := get(t, http.DefaultClient, url, nil)
	if first.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", first.StatusCode)
	}
	if got := first.Header.Get("x-hyperblur-cache"); got != "miss" {
		t.Fatalf("cache state = %q, want miss", got)
	}
	if !strings.Contains(bodyOf(t, first), "body text") {
		t.Fatal("page body missing rendered post content")
	}

	second := get(t, http.DefaultClient, url, nil)
	if got := second.Header.Get("x-hyperblur-cache"); got != "hit" {
		t.Fatalf("cache state = %q, want hit", got)
	}
	if calls := upstream.countContaining("hubs/cats/timeline"); calls != 1 {
		t.Fatalf("upstream called %d times for a cached page, want 1", calls)
	}
}

func TestPageCacheNegotiatesEncodingFromOneEntry(t *testing.T) {
	server, upstream := newTestServer(t)
	client := noCompressionClient()
	url := server.URL + "/tagged/encoding"

	gzipped := get(t, client, url, map[string]string{"Accept-Encoding": "gzip"})
	if got := gzipped.Header.Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding = %q, want gzip", got)
	}
	if got := gzipped.Header.Get("Vary"); !strings.Contains(got, "Accept-Encoding") {
		t.Fatalf("Vary = %q, want Accept-Encoding", got)
	}

	plain := get(t, client, url, map[string]string{"Accept-Encoding": "identity"})
	if got := plain.Header.Get("Content-Encoding"); got != "" {
		t.Fatalf("plain client got Content-Encoding %q", got)
	}
	plainLen := len(bodyOf(t, plain))
	gzipLen := len(bodyOf(t, gzipped))
	if gzipLen >= plainLen {
		t.Fatalf("gzip body (%d) is not smaller than plain (%d)", gzipLen, plainLen)
	}
	// Both were served from the same cache entry.
	if calls := upstream.countContaining("hubs/encoding/timeline"); calls != 1 {
		t.Fatalf("upstream called %d times, want 1 shared fill", calls)
	}
}

func TestPollResultsAreJSONValidatedAndCached(t *testing.T) {
	server, upstream := newTestServer(t)
	url := server.URL + "/api/v1/poll/staff/123/abcdef/results"

	first := get(t, http.DefaultClient, url, nil)
	if first.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", first.StatusCode)
	}
	if ct := first.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("Content-Type = %q, want JSON", ct)
	}
	if body := bodyOf(t, first); !strings.Contains(body, `"a1":7`) {
		t.Fatalf("body = %q, want upstream poll results", body)
	}

	get(t, http.DefaultClient, url, nil)
	if calls := upstream.countContaining("polls/staff/123/abcdef/results"); calls != 1 {
		t.Fatalf("upstream called %d times, want 1 with the poll cache", calls)
	}

	for _, bad := range []string{
		server.URL + "/api/v1/poll/staff/abc/abcdef/results",              // post id not numeric
		server.URL + "/api/v1/poll/staff/123/bad%2Fid/results",            // slash in poll id
		server.URL + "/api/v1/poll/x/1%2F..%2F..%2Fhealthz/1/results",     // traversal attempt
		server.URL + "/api/v1/poll/UPPER_CASE_IS_FINE/123/abcdef/results", // unknown blog shape
	} {
		resp := get(t, http.DefaultClient, bad, nil)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("GET %s = %d, want 400", bad, resp.StatusCode)
		}
		if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
			t.Fatalf("error Content-Type = %q, want JSON", ct)
		}
		if body := bodyOf(t, resp); !strings.Contains(body, "error") {
			t.Fatalf("error body = %q, want JSON error", body)
		}
	}
}

func TestMediaProxy(t *testing.T) {
	server, upstream := newTestServer(t)
	client := noCompressionClient()
	media := server.URL + "/tblr/media/64/a/b.jpg"

	resp := get(t, client, media, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); got != "image/jpeg" {
		t.Fatalf("Content-Type = %q", got)
	}
	if got := resp.Header.Get("Cache-Control"); got != "max-age=315360000" {
		t.Fatalf("Cache-Control = %q, want the upstream value", got)
	}
	if got := resp.Header.Get("ETag"); got != `"upstream-etag"` {
		t.Fatalf("ETag = %q, want upstream ETag", got)
	}
	if got := resp.Header.Get("Set-Cookie"); got != "" {
		t.Fatalf("upstream cookie leaked: %q", got)
	}
	if got := resp.Header.Get("Strict-Transport-Security"); got != "" {
		t.Fatalf("upstream HSTS leaked: %q", got)
	}
	if body := bodyOf(t, resp); body != string(mediaBody) {
		t.Fatalf("body = %q, want upstream bytes", body)
	}

	// Query strings must survive the rewrite and reach the CDN.
	withQuery := "/tblr/media/64/a/b.jpg?sig=abc123"
	get(t, client, server.URL+withQuery, nil)
	if calls := upstream.countContaining("64.media.tumblr.com/a/b.jpg?sig=abc123"); calls != 1 {
		t.Fatalf("upstream did not receive the query string (calls=%d)", calls)
	}

	// The download flag is hyperblur's own and must not be forwarded.
	download := get(t, client, media+"?download=1", nil)
	if got := download.Header.Get("Content-Disposition"); !strings.Contains(got, `filename="b.jpg"`) {
		t.Fatalf("Content-Disposition = %q, want an attachment filename", got)
	}
	if calls := upstream.countContaining("b.jpg?download=1"); calls != 0 {
		t.Fatal("download flag was forwarded upstream")
	}

	// Traversal attempts never reach the CDN.
	traversal := server.URL + "/tblr/media/64/a%2F..%2F..%2Fetc/passwd"
	if code := get(t, client, traversal, nil).StatusCode; code != http.StatusNotFound {
		t.Fatalf("traversal status = %d, want 404", code)
	}
	if calls := upstream.countContaining("etc/passwd"); calls != 0 {
		t.Fatal("traversal request reached upstream")
	}
}

// TestMediaForwardsRangeAndConditionals keeps video seeking and CDN
// revalidation working through the proxy.
func TestMediaForwardsRangeAndConditionals(t *testing.T) {
	server, upstream := newTestServer(t)
	client := noCompressionClient()
	media := server.URL + "/tblr/media/64/a/b.jpg"

	get(t, client, media, map[string]string{
		"Range":             "bytes=0-99",
		"If-None-Match":     `"upstream-etag"`,
		"If-Modified-Since": "Wed, 21 Oct 2026 07:28:00 GMT",
	})
	if calls := upstream.countContaining("b.jpg"); calls != 1 {
		t.Fatalf("upstream calls = %d, want 1", calls)
	}

	seen := upstream.lastRequest()
	for header, want := range map[string]string{
		"Range":             "bytes=0-99",
		"If-None-Match":     `"upstream-etag"`,
		"If-Modified-Since": "Wed, 21 Oct 2026 07:28:00 GMT",
	} {
		if got := seen.Get(header); got != want {
			t.Fatalf("forwarded %s = %q, want %q", header, got, want)
		}
	}
	if got := seen.Get("Referer"); got != "https://www.tumblr.com/" {
		t.Fatalf("Referer = %q, want the Tumblr referer", got)
	}
}

func TestSecurityHeadersAndStaticRoutes(t *testing.T) {
	server, _ := newTestServer(t)
	client := noCompressionClient()

	resp := get(t, client, server.URL+"/healthz", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/healthz = %d", resp.StatusCode)
	}
	if body := bodyOf(t, resp); !strings.Contains(body, "ok") {
		t.Fatalf("/healthz body = %q", body)
	}
	csp := resp.Header.Get("Content-Security-Policy")
	for _, want := range []string{"default-src 'none'", "frame-ancestors 'none'", "base-uri 'none'", "form-action 'self'"} {
		if !strings.Contains(csp, want) {
			t.Fatalf("CSP %q missing %q", csp, want)
		}
	}
	if got := resp.Header.Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("X-Frame-Options = %q, want DENY", got)
	}

	for _, route := range []string{"/robots.txt", "/manifest.webmanifest", "/sw.js", "/offline"} {
		if code := get(t, client, server.URL+route, nil).StatusCode; code != http.StatusOK {
			t.Fatalf("GET %s = %d, want 200", route, code)
		}
	}

	if code := get(t, client, server.URL+"/nosuchpage/nope", nil).StatusCode; code == http.StatusOK {
		t.Fatal("unknown path returned 200")
	}
}

func TestPrewarmFillsTheCache(t *testing.T) {
	upstream := newFakeUpstream()
	app := newTestApp(t, upstream)
	server := httptest.NewServer(app.Handler())
	t.Cleanup(server.Close)

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if app.pages.Len() > 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("prewarm did not fill the page cache")
}

func TestHEADRequestIsServedFromCache(t *testing.T) {
	server, _ := newTestServer(t)
	client := noCompressionClient()

	get(t, client, server.URL+"/tagged/head", nil)

	req, err := http.NewRequest(http.MethodHead, server.URL+"/tagged/head", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("HEAD: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("HEAD status = %d, want 200", resp.StatusCode)
	}
	if body, _ := io.ReadAll(resp.Body); len(body) != 0 {
		t.Fatalf("HEAD returned a body of %d bytes", len(body))
	}
}
