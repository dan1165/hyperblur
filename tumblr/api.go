package tumblr

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Explore post-type filters.
type ExplorePostType string

const (
	ExploreText   ExplorePostType = "text"
	ExplorePhotos ExplorePostType = "photos"
	ExploreGIFs   ExplorePostType = "gifs"
	ExploreQuotes ExplorePostType = "quotes"
	ExploreChats  ExplorePostType = "chats"
	ExploreAudio  ExplorePostType = "audio"
	ExploreVideo  ExplorePostType = "video"
	ExploreAsks   ExplorePostType = "asks"
)

// Post type filters used by search.
type PostTypeFilter string

const (
	FilterText   PostTypeFilter = "text"
	FilterPhoto  PostTypeFilter = "photo"
	FilterGIF    PostTypeFilter = "gif"
	FilterQuote  PostTypeFilter = "quote"
	FilterLink   PostTypeFilter = "link"
	FilterChat   PostTypeFilter = "chat"
	FilterAudio  PostTypeFilter = "audio"
	FilterVideo  PostTypeFilter = "video"
	FilterAnswer PostTypeFilter = "answer"
	FilterPoll   PostTypeFilter = "poll"
)

// ReblogNoteTypes selects the reblog note viewer filter.
type ReblogNoteTypes string

const (
	ReblogsWithComments        ReblogNoteTypes = "reblogs_with_comments"
	ReblogsWithContentComments ReblogNoteTypes = "reblogs_with_content_comments"
	ReblogsOnly                ReblogNoteTypes = "reblogs_only"
)

// ErrorKind classifies an API error for the web layer.
type ErrorKind int

const (
	ErrGeneric ErrorKind = iota
	ErrRestrictedTag
	ErrLoginRequired
	ErrPasswordRequired
	ErrBlogNotFound
	ErrNon200NorJSON
)

// APIError is a Tumblr API error.
type APIError struct {
	Kind         ErrorKind
	HTTPCode     int
	Message      string
	Details      string
	InternalCode int
}

func (e *APIError) Error() string {
	message := fmt.Sprintf("Tumblr has returned an error response\nHTTP Code: %d\nMessage: %s", e.HTTPCode, e.Message)
	if e.Details != "" {
		message += "\nDetails: " + e.Details
	}
	if e.InternalCode != 0 {
		message += fmt.Sprintf("\nError Code: %d", e.InternalCode)
	}
	return message
}

// Field-selection query values, mirroring request_config.py.
const (
	ExploreBlogInfoFields   = "?advertiser_name,?avatar,?blog_view_url,?can_be_booped,?can_be_followed,?can_show_badges,?description_npf,?followed,?is_adult,?is_member,name,?primary,?theme,?title,?tumblrmart_accessories,url,?uuid,?ask,?can_submit,?can_subscribe,?is_blocked_from_primary,?is_blogless_advertiser,?is_password_protected,?share_following,?share_likes,?subscribed"
	SearchBlogInfoFields    = "?advertiser_name,?avatar,?blog_view_url,?can_be_booped,?can_be_followed,?can_show_badges,?description_npf,?followed,?is_adult,?is_member,name,?primary,?theme,?title,?tumblrmart_accessories,url,?uuid,?share_following,?share_likes,?ask"
	PostBlogInfoFields      = "?advertiser_name,?avatar,?blog_view_url,?can_be_booped,?can_be_followed,?can_show_badges,?description_npf,?followed,?is_adult,?is_member,name,?primary,?theme,?title,?tumblrmart_accessories,url,?uuid,?share_likes,?share_following,?can_subscribe,?subscribed,?allow_search_indexing,?ask,?can_submit,?is_blocked_from_primary,?analytics_url,?is_hidden_from_blog_network"
	BlogPostsBlogInfoFields = "?advertiser_name,?avatar,?blog_view_url,?can_be_booped,?can_be_followed,?can_show_badges,?description_npf,?followed,?is_adult,?is_member,name,?primary,?theme,?title,?tumblrmart_accessories,url,?uuid,?ask,?can_submit,?can_subscribe,?is_blocked_from_primary,?is_blogless_advertiser,?is_password_protected,?share_following,?share_likes,?subscribed,?admin,?can_message,?ask_page_title,?analytics_url,?top_tags,?allow_search_indexing,?is_hidden_from_blog_network,?should_show_gift,?should_show_tumblrmart_gift"
)

const (
	defaultAuthorizationToken = "aIcXSOoTtqrzR8L8YEIOmBeW94c3FmbSNSWAUbxsny9KKx5VFh"
	authorizationEnvVar       = "OPENBLUR_TUMBLR_API_TOKEN"
	maxRetries                = 3
	retryBackoff              = 500 * time.Millisecond
)

// API is a Tumblr API client.
type API struct {
	client  *http.Client
	headers map[string]string
}

// DefaultUserAgent is the user agent openblur sends to Tumblr.
const DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:136.0) Gecko/20100101 Firefox/136.0"

// NewAPI builds a Tumblr API client, using a custom token from the
// environment when one is set.
func NewAPI(timeout time.Duration) *API {
	token := strings.TrimSpace(os.Getenv(authorizationEnvVar))
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		token = strings.TrimSpace(token[len("bearer "):])
	}
	if token == "" {
		token = defaultAuthorizationToken
	}

	return &API{
		client: &http.Client{Timeout: timeout},
		headers: map[string]string{
			"accept":        "application/json;format=camelcase",
			"user-agent":    DefaultUserAgent,
			"te":            "trailers",
			"connection":    "keep-alive",
			"referer":       "https://www.tumblr.com/",
			"authorization": "Bearer " + token,
		},
	}
}

// getJSON requests an endpoint, retrying transient network failures.
func (a *API) getJSON(endpoint string, params url.Values) (map[string]any, error) {
	target := "https://www.tumblr.com/api/v2/" + endpoint
	if len(params) > 0 {
		target += "?" + params.Encode()
	}

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		result, retryable, err := a.requestJSON(target)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if !retryable || attempt >= maxRetries {
			break
		}
		time.Sleep(retryBackoff * time.Duration(attempt))
	}
	return nil, lastErr
}

func (a *API) requestJSON(target string) (map[string]any, bool, error) {
	request, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return nil, false, err
	}
	for key, value := range a.headers {
		request.Header.Set(key, value)
	}

	response, err := a.client.Do(request)
	if err != nil {
		// Network-level failure: retryable.
		return nil, true, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, true, err
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		if response.StatusCode != http.StatusOK {
			return nil, false, &APIError{
				Kind:     ErrNon200NorJSON,
				HTTPCode: response.StatusCode,
				Message:  fmt.Sprintf("Tumblr returned a non-200 status code (%d)", response.StatusCode),
			}
		}
		return nil, false, fmt.Errorf("failed to parse JSON response from Tumblr: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		return nil, false, mapErrorResponse(result, response.StatusCode)
	}

	return result, false, nil
}

func mapErrorResponse(result map[string]any, httpStatus int) *APIError {
	meta := obj(result["meta"])
	message := str(meta["msg"])
	if message == "" {
		message = "Unknown error response from Tumblr"
	}
	code := integer(meta["status"])
	if code == 0 {
		code = httpStatus
	}

	var internalCode int
	var details string
	if errorsList := slice(result["errors"]); len(errorsList) > 0 {
		first := obj(errorsList[0])
		details = str(first["detail"])
		internalCode = integer(first["code"])
	}

	kind := ErrGeneric
	switch internalCode {
	case 13001:
		kind = ErrRestrictedTag
	case 4012:
		kind = ErrLoginRequired
	case 4013:
		kind = ErrPasswordRequired
	case 0:
		kind = ErrBlogNotFound
	}

	return &APIError{Kind: kind, HTTPCode: code, Message: message, Details: details, InternalCode: internalCode}
}

// ---------------------------------------------------------------------------
// Endpoints
// ---------------------------------------------------------------------------

// ExploreTrending requests /explore/trending.
func (a *API) ExploreTrending(continuation string) (map[string]any, error) {
	params := url.Values{"reblog_info": {"true"}, "fields[blogs]": {ExploreBlogInfoFields}}
	if continuation != "" {
		params.Set("cursor", continuation)
	}
	return a.getJSON("explore/trending", params)
}

// ExploreToday requests /explore/home/today.
func (a *API) ExploreToday(continuation string) (map[string]any, error) {
	params := url.Values{"fields[blogs]": {ExploreBlogInfoFields}, "reblog_info": {"true"}}
	if continuation != "" {
		params.Set("cursor", continuation)
	}
	return a.getJSON("explore/home/today", params)
}

// ExplorePost requests /explore/posts/<type>.
func (a *API) ExplorePost(postType ExplorePostType, continuation string) (map[string]any, error) {
	params := url.Values{"reblog_info": {"true"}, "fields[blogs]": {ExploreBlogInfoFields}}
	if continuation != "" {
		params.Set("cursor", continuation)
	}
	return a.getJSON("explore/posts/"+string(postType), params)
}

// TimelineSearch requests /timeline/search.
func (a *API) TimelineSearch(query, continuation string, latest bool, days int, postTypeFilter PostTypeFilter) (map[string]any, error) {
	mode := "top"
	if latest {
		mode = "recent"
	}
	params := url.Values{
		"limit":          {"20"},
		"days":           {fmt.Sprint(days)},
		"query":          {query},
		"mode":           {mode},
		"reblog_info":    {"true"},
		"timeline_type":  {"post"},
		"skip_component": {"related_tags,blog_search"},
	}

	if postTypeFilter != "" {
		params.Set("post_type_filter", string(postTypeFilter))
	}
	params.Set("fields[blogs]", SearchBlogInfoFields)

	if continuation != "" {
		params.Set("cursor", continuation)
	}
	return a.getJSON("timeline/search", params)
}

// HubsTimeline requests /hubs/<tag>/timeline.
func (a *API) HubsTimeline(tag string, continuation string, latest bool) (map[string]any, error) {
	sort := "top"
	if latest {
		sort = "recent"
	}
	params := url.Values{
		"fields[blogs]": {ExploreBlogInfoFields},
		"sort":          {sort},
		"limit":         {"14"},
	}
	if continuation != "" {
		params.Set("hub_name", tag)
		params.Set("rawurldecode", "1")
		params.Set("skip_header", "1")
		params.Set("cursor", continuation)
	}
	return a.getJSON("hubs/"+url.PathEscape(tag)+"/timeline", params)
}

// BlogPosts requests /blog/<name>/posts.
func (a *API) BlogPosts(blogName, continuation, tag, offset, limit string) (map[string]any, error) {
	params := url.Values{
		"fields[blogs]":        {BlogPostsBlogInfoFields},
		"npf":                  {"true"},
		"reblog_info":          {"true"},
		"include_pinned_posts": {"true"},
	}
	if tag != "" {
		params.Set("tag", tag)
	}
	if offset != "" {
		params.Set("offset", offset)
	}
	if limit != "" {
		params.Set("limit", limit)
	}
	if continuation != "" {
		params.Set("tumblelog", blogName)
		params.Set("page_number", continuation)
	}
	return a.getJSON("blog/"+url.PathEscape(blogName)+"/posts", params)
}

// BlogSearch requests /blog/<name>/search/<query>.
func (a *API) BlogSearch(blogName, query, continuation string) (map[string]any, error) {
	escapedBlog := url.PathEscape(blogName)
	params := url.Values{
		"reblog_info":   {"true"},
		"fields[blogs]": {ExploreBlogInfoFields},
		"npf":           {"true"},
		"sort":          {"CREATED_DESC"},
	}
	if continuation != "" {
		params.Set("tumblelog", blogName)
		params.Set("query", query)
		params.Set("rawurldecode", "1")
		params.Set("cursor", continuation)
	}
	return a.getJSON("blog/"+escapedBlog+"/search/"+url.PathEscape(query), params)
}

// BlogPost requests /blog/<name>/posts/<id>/permalink.
func (a *API) BlogPost(blogName, postID string) (map[string]any, error) {
	params := url.Values{"fields[blogs]": {PostBlogInfoFields}, "reblog_info": {"true"}}
	return a.getJSON("blog/"+url.PathEscape(blogName)+"/posts/"+postID+"/permalink", params)
}

// BlogPostReplies requests /blog/<id>/post/<id>/replies.
func (a *API) BlogPostReplies(blogID, postID, afterID string, latest bool) (map[string]any, error) {
	sort := "asc"
	if latest {
		sort = "desc"
	}
	var params url.Values
	if afterID == "" {
		params = url.Values{
			"mode":             {"replies"},
			"sort":             {sort},
			"pin_preview_note": {"false"},
			"fields[blogs]":    {"avatar,theme,name"},
		}
	} else {
		params = url.Values{"after": {afterID}, "sort": {sort}}
	}
	return a.getJSON("blog/"+url.PathEscape(blogID)+"/post/"+postID+"/replies", params)
}

// BlogPostNotesTimeline requests /blog/<id>/post/<id>/notes/timeline.
func (a *API) BlogPostNotesTimeline(blogID, postID string, mode ReblogNoteTypes, latest bool, beforeTimestamp string) (map[string]any, error) {
	var params url.Values
	if beforeTimestamp != "" {
		params = url.Values{
			"id":               {postID},
			"mode":             {string(mode)},
			"before_timestamp": {beforeTimestamp},
		}
	} else {
		sort := "desc"
		if latest {
			sort = "asc"
		}
		params = url.Values{
			"mode":             {string(mode)},
			"sort":             {sort},
			"pin_preview_note": {"false"},
			"fields[blogs]":    {"avatar,theme,name"},
		}
	}
	return a.getJSON("blog/"+url.PathEscape(blogID)+"/post/"+postID+"/notes/timeline", params)
}

// BlogNotes requests /blog/<id>/notes.
func (a *API) BlogNotes(blogID, postID string, latest, returnLikes bool, beforeTimestamp string) (map[string]any, error) {
	mode := "likes"
	if !returnLikes {
		mode = "reblogs_only"
	}
	var params url.Values
	if beforeTimestamp != "" {
		params = url.Values{"mode": {mode}, "id": {postID}, "before_timestamp": {beforeTimestamp}}
	} else {
		sort := "asc"
		if latest {
			sort = "desc"
		}
		params = url.Values{"id": {postID}, "mode": {mode}, "sort": {sort}}
	}
	return a.getJSON("blog/"+url.PathEscape(blogID)+"/notes", params)
}

// PollResults requests /polls/<blog>/<post>/<poll>/results.
func (a *API) PollResults(blogName, postID, pollID string) (map[string]any, error) {
	return a.getJSON("polls/"+url.PathEscape(blogName)+"/"+postID+"/"+pollID+"/results", nil)
}
