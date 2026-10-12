// Package web implements hyperblur's HTTP server and pages.
package web

import (
	"log"
	"net"
	"net/http"
	"time"

	"github.com/dan1165/hyperblur/npf"
	"github.com/dan1165/hyperblur/render"
	"github.com/dan1165/hyperblur/tumblr"
)

// App holds the shared server state.
type App struct {
	API    *tumblr.API
	Media  *http.Client
	Logger *log.Logger
}

// NewApp builds the application, creating the Tumblr API client.
func NewApp(logger *log.Logger) *App {
	transport := newTransport()
	return &App{
		API:    tumblr.NewAPI(transport, 10*time.Second),
		Media:  &http.Client{Transport: transport, Timeout: 30 * time.Second},
		Logger: logger,
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
func (a *App) FormatNPF(content, layout []any, blogName, postID string, fetchPolls bool) (*render.RenderError, string) {
	params := render.Params{
		BlogName: blogName,
		PostID:   postID,
	}
	if fetchPolls && blogName != "" && postID != "" {
		params.PollCallback = a.pollCallback(blogName, postID)
	}
	return render.FormatNPF(content, layout, params)
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
