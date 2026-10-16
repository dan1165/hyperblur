package web

import (
	"bytes"
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// pageResponse is a cached HTML response: the raw body plus a precomputed gzip
// variant, so any client encoding is served without recompressing.
type pageResponse struct {
	status int
	header http.Header
	raw    []byte
	gz     []byte
	stored time.Time
	ttl    time.Duration
	stale  time.Duration
}

func (p *pageResponse) age() time.Duration { return time.Since(p.stored) }
func (p *pageResponse) fresh() bool        { return p.age() <= p.ttl }
func (p *pageResponse) usable() bool       { return p.age() <= p.ttl+p.stale }

// write replays the cached response, choosing the encoding for this client.
func (p *pageResponse) write(w http.ResponseWriter, r *http.Request, state string) {
	h := w.Header()
	for key, values := range p.header {
		for _, value := range values {
			h.Add(key, value)
		}
	}
	h.Del("Content-Length")

	body := p.raw
	if len(p.gz) > 0 && acceptsGzip(r) {
		body = p.gz
		h.Set("Content-Encoding", "gzip")
	}
	h.Set("Vary", "Accept-Encoding")
	h.Set("Content-Length", strconv.Itoa(len(body)))
	if p.status == http.StatusOK {
		h.Set("x-hyperblur-cache", state)
	}
	w.WriteHeader(p.status)
	if p.status != http.StatusNotModified {
		_, _ = w.Write(body)
	}
}

// recorder captures a handler's response so it can be cached and/or replayed.
// It is only used for cacheable paths, which never stream.
type recorder struct {
	header  http.Header
	status  int
	written bool
	buf     bytes.Buffer
}

func newRecorder() *recorder { return &recorder{header: make(http.Header)} }

func (r *recorder) Header() http.Header { return r.header }

func (r *recorder) WriteHeader(status int) {
	if r.written {
		return
	}
	r.status = status
	r.written = true
}

func (r *recorder) Write(b []byte) (int, error) {
	if !r.written {
		r.status = http.StatusOK
		r.written = true
	}
	return r.buf.Write(b)
}

func (r *recorder) response(ttl, stale time.Duration) *pageResponse {
	status := r.status
	if status == 0 {
		status = http.StatusOK
	}
	// Own the bytes so the recorder can be discarded.
	raw := append([]byte(nil), r.buf.Bytes()...)
	resp := &pageResponse{
		status: status,
		header: r.header,
		raw:    raw,
		stored: time.Now(),
		ttl:    ttl,
		stale:  stale,
	}
	if status == http.StatusOK {
		resp.gz = gzipCompress(raw)
	}
	return resp
}

// cacheKey is the page's URL. Encoding is negotiated at serve time.
func cacheKey(r *http.Request) string {
	if r.URL.RawQuery == "" {
		return r.URL.Path
	}
	return r.URL.Path + "?" + r.URL.RawQuery
}

func cacheablePath(path string) bool {
	switch {
	case strings.HasPrefix(path, "/assets/"),
		strings.HasPrefix(path, "/tblr/"),
		strings.HasPrefix(path, "/api/"),
		strings.HasPrefix(path, "/at/"):
		return false
	}
	switch path {
	case "/robots.txt", "/manifest.webmanifest", "/sw.js", "/healthz":
		return false
	}
	return true
}

func acceptsGzip(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")
}

// pageCache serves cached HTML pages, refreshes stale ones in the background,
// and collapses concurrent misses of the same page with singleflight.
func (a *App) pageCache(next http.Handler) http.Handler {
	if !a.caching {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if (r.Method != http.MethodGet && r.Method != http.MethodHead) || !cacheablePath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		key := cacheKey(r)
		if item, ok := a.pages.Get(key); ok {
			if !item.usable() {
				a.pages.Remove(key)
			} else {
				state := "hit"
				if !item.fresh() {
					// Serve stale immediately; refresh in the background.
					state = "stale"
					a.refresh(key, r)
				}
				item.write(w, r, state)
				return
			}
		}

		value, _, _ := a.sf.Do(key, func() (any, error) {
			if item, ok := a.pages.Get(key); ok && item.usable() {
				return item, nil
			}
			rec := newRecorder()
			next.ServeHTTP(rec, r)
			resp := rec.response(a.cacheTTL, a.cacheStale)
			if resp.status == http.StatusOK {
				a.pages.Add(key, resp)
			}
			return resp, nil
		})

		if resp, ok := value.(*pageResponse); ok && resp != nil {
			resp.write(w, r, "miss")
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
	})
}

// refresh re-runs the request in the background and updates the cache entry.
// Concurrent refreshes of the same key are collapsed by singleflight.
func (a *App) refresh(key string, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	clone := r.Clone(ctx)
	go func() {
		defer cancel()
		_, _, _ = a.refreshSF.Do(key, func() (any, error) {
			rec := newRecorder()
			a.inner.ServeHTTP(rec, clone)
			if resp := rec.response(a.cacheTTL, a.cacheStale); resp.status == http.StatusOK {
				a.pages.Add(key, resp)
			}
			return nil, nil
		})
	}()
}
