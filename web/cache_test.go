package web

import (
	"net/http"
	"net/url"
	"testing"
)

func newTestRequest(rawURL string, headers map[string]string) *http.Request {
	parsed, _ := url.Parse(rawURL)
	req := &http.Request{Method: http.MethodGet, URL: parsed, Header: http.Header{}}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return req
}

func TestCacheKeyIgnoresEncoding(t *testing.T) {
	plain := cacheKey(newTestRequest("/staff", nil))
	gzipped := cacheKey(newTestRequest("/staff", map[string]string{"Accept-Encoding": "gzip, deflate"}))
	if plain != gzipped {
		t.Fatalf("cache key should not depend on encoding: %q vs %q", plain, gzipped)
	}
}

func TestCacheKeyIncludesQuery(t *testing.T) {
	withQuery := cacheKey(newTestRequest("/staff?page=2", nil))
	if withQuery != "/staff?page=2" {
		t.Fatalf("cacheKey = %q, want /staff?page=2", withQuery)
	}
	if cacheKey(newTestRequest("/staff", nil)) == withQuery {
		t.Fatal("paged URL must not share a cache key with page 1")
	}
}

func TestCacheablePath(t *testing.T) {
	cacheable := []string{"/staff", "/explore/trending", "/search/cats", "/offline", "/staff/123/slug"}
	for _, path := range cacheable {
		if !cacheablePath(path) {
			t.Fatalf("cacheablePath(%q) = false, want true", path)
		}
	}

	notCacheable := []string{"/assets/js/post.js", "/tblr/media/64/a.jpg", "/api/v1/poll/a/1/b/results", "/at/abc", "/healthz", "/sw.js", "/robots.txt"}
	for _, path := range notCacheable {
		if cacheablePath(path) {
			t.Fatalf("cacheablePath(%q) = true, want false", path)
		}
	}
}

func TestAcceptsGzip(t *testing.T) {
	if !acceptsGzip(newTestRequest("/", map[string]string{"Accept-Encoding": "gzip, deflate, br"})) {
		t.Fatal("gzip client not detected")
	}
	if acceptsGzip(newTestRequest("/", nil)) {
		t.Fatal("plain client detected as gzip")
	}
}
