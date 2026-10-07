package tumblr

import (
	"fmt"
	"time"
)

// ---------------------------------------------------------------------------
// generic JSON helpers
// ---------------------------------------------------------------------------

func obj(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func str(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

func integer(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	}
	return 0
}

func boolean(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t != ""
	case float64:
		return t != 0
	case nil:
		return false
	}
	return false
}

func slice(v any) []any {
	l, _ := v.([]any)
	return l
}

func stringSlice(v any) []string {
	l := slice(v)
	out := make([]string, 0, len(l))
	for _, item := range l {
		out = append(out, str(item))
	}
	return out
}

func avatarSlice(v any) []Avatar {
	l := slice(v)
	out := make([]Avatar, 0, len(l))
	for _, item := range l {
		a := obj(item)
		out = append(out, Avatar{URL: str(a["url"]), Width: integer(a["width"]), Height: integer(a["height"])})
	}
	return out
}

func avatarMap(v any) map[string]string {
	out := map[string]string{}
	switch t := v.(type) {
	case map[string]any:
		for key, value := range t {
			out[key] = str(value)
		}
	case []any:
		for _, item := range t {
			entry := obj(item)
			if key := str(entry["width"]); key != "" {
				out[key] = str(entry["url"])
			}
		}
	}
	return out
}

func timeFromUnix(v any) *time.Time {
	if v == nil {
		return nil
	}
	switch n := v.(type) {
	case float64:
		t := time.Unix(int64(n), 0).UTC()
		return &t
	case int:
		t := time.Unix(int64(n), 0).UTC()
		return &t
	case int64:
		t := time.Unix(n, 0).UTC()
		return &t
	}
	return nil
}

func digDict(target map[string]any, keys ...string) any {
	var cur any = target
	for _, key := range keys {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = m[key]
	}
	return cur
}

func optionalInt(v any) *int {
	if v == nil {
		return nil
	}
	n := integer(v)
	return &n
}

// ---------------------------------------------------------------------------
// Blog
// ---------------------------------------------------------------------------

func parseBlog(target map[string]any) *Blog {
	theme := obj(target["theme"])
	return &Blog{
		Name:                  str(target["name"]),
		Avatar:                avatarSlice(target["avatar"]),
		Title:                 str(target["title"]),
		DescriptionNPF:        slice(target["descriptionNpf"]),
		Banner:                str(theme["headerImageFocused"]),
		Active:                target["active"] == nil || boolean(target["active"]),
		RequiresAccountToView: boolean(target["isHiddenFromBlogNetwork"]),
	}
}

func parseLimitedBlog(target map[string]any) *Blog {
	active := true
	if v, ok := target["active"]; ok {
		active = boolean(v)
	}
	theme := obj(target["theme"])
	return &Blog{
		Name:           str(target["name"]),
		Avatar:         avatarSlice(target["avatar"]),
		Title:          str(target["title"]),
		DescriptionNPF: slice(target["descriptionNpf"]),
		Banner:         str(theme["headerImageFocused"]),
		Active:         active,
	}
}

// ---------------------------------------------------------------------------
// Post
// ---------------------------------------------------------------------------

func parsePost(target map[string]any) *Post {
	blog := parseBlog(obj(target["blog"]))

	// Pick the first non-empty note tab, defaulting to replies.
	defaultTab := "replies"
	for _, tab := range []struct {
		name  string
		count *int
	}{
		{"replies", optionalInt(target["replyCount"])},
		{"reblogs", optionalInt(target["reblogCount"])},
		{"likes", optionalInt(target["likeCount"])},
	} {
		if tab.count != nil && *tab.count > 0 {
			defaultTab = tab.name
			break
		}
	}

	isAdvertisement := target["advertiserId"] != nil || target["adId"] != nil || target["adProviderId"] != nil

	var trails []PostTrail
	for _, rawTrail := range slice(target["trail"]) {
		trailPost := obj(rawTrail)

		var trailBlog *Blog
		var brokenBlog *BrokenBlog
		isBroken := false

		if rawBlog, ok := trailPost["blog"]; ok && rawBlog != nil {
			trailBlog = parseBlog(obj(rawBlog))
		} else {
			brokenBlog = &BrokenBlog{Name: str(obj(trailPost["brokenBlog"])["name"])}
			isBroken = true
		}

		var trailPostID string
		var trailDate *time.Time
		if postData := obj(trailPost["post"]); postData != nil && !isBroken {
			trailPostID = str(postData["id"])
			trailDate = timeFromUnix(postData["timestamp"])
		}

		trails = append(trails, PostTrail{
			ID:         trailPostID,
			Blog:       trailBlog,
			BrokenBlog: brokenBlog,
			Date:       trailDate,
			Content:    slice(trailPost["content"]),
			Layout:     slice(trailPost["layout"]),
		})
	}

	var reblogFrom, reblogRoot *ReblogAttribution
	if rebloggedFromID := str(target["rebloggedFromId"]); rebloggedFromID != "" {
		reblogFrom = &ReblogAttribution{
			PostID:   rebloggedFromID,
			PostURL:  str(target["parentPostUrl"]),
			BlogName: str(target["rebloggedFromName"]),
		}
		if rootID := str(target["rebloggedRootId"]); rootID != "" {
			reblogRoot = &ReblogAttribution{
				PostID:   rootID,
				PostURL:  str(target["rebloggedRootUrl"]),
				BlogName: str(target["rebloggedRootName"]),
			}
		}
	}

	return &Post{
		Blog:                 blog,
		ID:                   str(target["id"]),
		IsAdvertisement:      isAdvertisement,
		Slug:                 str(target["slug"]),
		Date:                 timeFromUnix(target["timestamp"]),
		Tags:                 stringSlice(target["tags"]),
		Content:              slice(target["content"]),
		Layout:               slice(target["layout"]),
		Trail:                trails,
		NoteCount:            optionalInt(target["noteCount"]),
		DefaultNoteViewerTab: defaultTab,
		ReblogFrom:           reblogFrom,
		ReblogRoot:           reblogRoot,
	}
}

// ---------------------------------------------------------------------------
// Notes
// ---------------------------------------------------------------------------

func parseReplyNote(target map[string]any) *ReplyNote {
	return &ReplyNote{
		ReplyID: str(target["replyId"]),
		Date:    timeFromUnix(target["timestamp"]),
		Content: slice(target["content"]),
		Layout:  slice(target["layout"]),
		Blog:    parseLimitedBlog(obj(target["blog"])),
	}
}

func parseReblogNote(target map[string]any) *ReblogNote {
	return &ReblogNote{
		ID:      str(target["postId"]),
		Blog:    parseLimitedBlog(obj(target["blog"])),
		Content: slice(target["content"]),
		Layout:  slice(target["content"]),
		Tags:    stringSlice(target["tags"]),
		Date:    timeFromUnix(target["timestamp"]),
	}
}

func parseSimpleReblogNote(target map[string]any) *ReblogNote {
	avatarMap := obj(target["avatarUrl"])
	var avatars []Avatar
	for _, v := range avatarMap {
		if s, ok := v.(string); ok {
			avatars = append(avatars, Avatar{URL: s})
		}
	}
	blog := &Blog{
		Name:   str(target["blogName"]),
		Avatar: avatars,
		Title:  str(target["blogTitle"]),
		Active: true,
	}
	return &ReblogNote{
		ID:      str(target["postId"]),
		Blog:    blog,
		Content: []any{},
		Layout:  []any{},
		Tags:    stringSlice(target["tags"]),
		Date:    timeFromUnix(target["timestamp"]),
	}
}

func parseLikeNote(target map[string]any) *LikeNote {
	return &LikeNote{
		BlogName:  str(target["blogName"]),
		BlogTitle: str(target["blogTitle"]),
		Date:      timeFromUnix(target["timestamp"]),
		Avatar:    avatarMap(target["avatarUrl"]),
	}
}

func parseSignpost(target map[string]any) *Signpost {
	return &Signpost{
		Title:       str(obj(target["display"])["title"]),
		Description: str(digDict(target, "resources", "description")),
	}
}

// ---------------------------------------------------------------------------
// Item dispatcher
// ---------------------------------------------------------------------------

// ItemParser converts one raw element into a model, or nil if it does not match.
type ItemParser func(map[string]any) any

// ParseItem tries each parser in order and returns the first match.
func ParseItem(element map[string]any, parsers ...ItemParser) any {
	for _, parser := range parsers {
		if parsed := parser(element); parsed != nil {
			return parsed
		}
	}
	return nil
}

// ParsePostItem parses a raw element that must be a post.
func ParsePostItem(element map[string]any) any {
	if str(element["objectType"]) == "post" {
		return parsePost(element)
	}
	return nil
}

// ParseSignpostItem parses a raw element that must be a signpost.
func ParseSignpostItem(element map[string]any) any {
	if str(element["objectType"]) == "signpost_cta" {
		return parseSignpost(element)
	}
	return nil
}

// ParseReplyNoteItem parses a raw element that must be a reply note.
func ParseReplyNoteItem(element map[string]any) any {
	if str(element["type"]) != "reply" {
		return nil
	}
	return parseReplyNote(element)
}

// ParseReblogNoteItem parses a raw element that must be a reblog note.
func ParseReblogNoteItem(element map[string]any) any {
	if str(element["type"]) != "reblog" {
		return nil
	}
	if _, ok := element["blogName"]; ok {
		return parseSimpleReblogNote(element)
	}
	return parseReblogNote(element)
}

// ParseLikeNoteItem parses a raw element that must be a like note.
func ParseLikeNoteItem(element map[string]any) any {
	if str(element["type"]) != "like" {
		return nil
	}
	return parseLikeNote(element)
}
