package web

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/dan1165/openblur/tumblr"
)

var mediaRequestHeaders = map[string]string{
	"user-agent":      tumblr.DefaultUserAgent,
	"accept":          "image/avif,image/webp,image/png,image/svg+xml,image/*;q=0.8,*/*;q=0.5",
	"accept-language": "en-US,en;q=0.5",
	"connection":      "keep-alive",
	"te":              "trailers",
	"referer":         "https://www.tumblr.com/",
}

var forwardedRequestHeaders = []string{"range", "if-range", "if-none-match", "if-modified-since"}

// Headers openblur never forwards from upstream media responses.
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
	path := r.PathValue("path")
	if !validMediaPath(path) {
		http.NotFound(w, r)
		return
	}

	origin := "https://" + cdn + ".media.tumblr.com/" + quotePath(path)
	if !downloadRequested(r) {
		http.Redirect(w, r, origin, http.StatusFound)
		return
	}

	extra := map[string]string(nil)
	switch cdn {
	case "ve", "va":
		extra = map[string]string{"accept": videoAccept}
	}
	a.streamMedia(w, r, origin, extra, buildContentDisposition(path))
}

func (a *App) handleMediaAudio(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("path")
	if !validMediaPath(path) {
		http.NotFound(w, r)
		return
	}
	origin := "https://a.tumblr.com/" + quotePath(path)
	if !downloadRequested(r) {
		http.Redirect(w, r, origin, http.StatusFound)
		return
	}
	a.streamMedia(w, r, origin, map[string]string{"accept": audioAccept}, buildContentDisposition(path))
}

func (a *App) handleMediaAssets(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("path")
	if !validMediaPath(path) {
		http.NotFound(w, r)
		return
	}
	origin := "https://assets.tumblr.com/" + quotePath(path)
	if !downloadRequested(r) {
		http.Redirect(w, r, origin, http.StatusFound)
		return
	}
	a.streamMedia(w, r, origin, nil, buildContentDisposition(path))
}

func (a *App) handleMediaStatic(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("path")
	if !validMediaPath(path) {
		http.NotFound(w, r)
		return
	}
	origin := "https://static.tumblr.com/" + quotePath(path)
	if !downloadRequested(r) {
		http.Redirect(w, r, origin, http.StatusFound)
		return
	}
	a.streamMedia(w, r, origin, nil, buildContentDisposition(path))
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
		a.messageError(w, r, http.StatusBadGateway, a.translate("openblur_error_invalid_internal_tumblr_redirect"), "")
		return
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusMovedPermanently {
		if location, err := url.Parse(response.Header.Get("location")); err == nil && strings.HasPrefix(location.Path, "/") {
			http.Redirect(w, r, location.Path, http.StatusFound)
			return
		}
	}
	a.messageError(w, r, http.StatusBadGateway, a.translate("openblur_error_invalid_internal_tumblr_redirect"), "")
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
	const unreserved = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~/"
	var b strings.Builder
	for i := 0; i < len(path); i++ {
		c := path[i]
		if strings.IndexByte(unreserved, c) >= 0 {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
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
