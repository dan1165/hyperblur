// Package web implements openblur's HTTP server and pages.
package web

import (
	"log"
	"net/http"
	"time"

	"github.com/dan1165/openblur/npf"
	"github.com/dan1165/openblur/render"
	"github.com/dan1165/openblur/tumblr"
)

// App holds the shared server state.
type App struct {
	API    *tumblr.API
	Media  *http.Client
	Logger *log.Logger
}

// NewApp builds the application, creating the Tumblr API client.
func NewApp(logger *log.Logger) *App {
	return &App{
		API:    tumblr.NewAPI(10 * time.Second),
		Media:  &http.Client{Timeout: 30 * time.Second},
		Logger: logger,
	}
}

// FormatNPF renders post content and returns an optional render error.
func (a *App) FormatNPF(content, layout []any, blogName, postID string, fetchPolls, expand bool) (*render.RenderError, string) {
	params := render.Params{
		ExpandPosts: expand,
		BlogName:    blogName,
		PostID:      postID,
	}
	if fetchPolls && blogName != "" && postID != "" {
		params.PollCallback = a.pollCallback(blogName, postID)
	}
	return render.FormatNPF(content, layout, params)
}

func (a *App) pollCallback(blog, postID string) npf.PollCallback {
	return func(pollID string, _ int64) (*npf.PollResults, error) {
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
		return &npf.PollResults{Timestamp: int64(toFloat(response["timestamp"])), Results: results}, nil
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
