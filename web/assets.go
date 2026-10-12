package web

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/dan1165/hyperblur/assets"
)

// assetEntry is a precomputed embedded file, gzipped when it is compressible.
type assetEntry struct {
	body  []byte
	gz    []byte
	etag  string
	ctype string
}

// loadAssets reads every embedded asset once, at startup.
func loadAssets() map[string]*assetEntry {
	out := map[string]*assetEntry{}
	_ = fs.WalkDir(assets.FS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		body, err := fs.ReadFile(assets.FS, p)
		if err != nil {
			return nil
		}

		sum := sha256.Sum256(body)
		entry := &assetEntry{
			body:  body,
			etag:  `"` + hex.EncodeToString(sum[:16]) + `"`,
			ctype: contentType(p),
		}
		if compressible(entry.ctype) && len(body) > 512 {
			var buf bytes.Buffer
			writer, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
			_, _ = writer.Write(body)
			_ = writer.Close()
			entry.gz = buf.Bytes()
		}
		out[p] = entry
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
