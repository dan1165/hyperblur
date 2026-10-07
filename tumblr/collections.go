package tumblr

// ParseTimeline parses an explore/search/tag API response.
func ParseTimeline(initialData map[string]any) *Timeline {
	target := obj(initialData["timeline"])
	if target == nil {
		return nil
	}

	var elements []any
	var signposts []Signpost
	for _, element := range slice(target["elements"]) {
		parsed := ParseItem(obj(element), ParsePostItem, ParseSignpostItem)
		switch value := parsed.(type) {
		case *Signpost:
			signposts = append(signposts, *value)
		case *Post:
			elements = append(elements, value)
		}
	}

	return &Timeline{
		Elements:  elements,
		Signposts: signposts,
		Next:      parseCursor(target),
	}
}

// ParseBlogTimeline parses a blog API response.
func ParseBlogTimeline(initialData map[string]any, isSearch bool) *BlogTimeline {
	if _, ok := initialData["blog"]; !ok {
		return nil
	}

	posts, totalRawPosts := parsePosts(initialData)

	if isSearch {
		var blogInfo *Blog
		if len(posts) > 0 {
			blogInfo = posts[0].Blog
		}
		return &BlogTimeline{
			BlogInfo:   blogInfo,
			Posts:      posts,
			TotalPosts: &totalRawPosts,
			Next:       parseCursor(initialData),
		}
	}

	return &BlogTimeline{
		BlogInfo:   parseBlog(obj(initialData["blog"])),
		Posts:      posts,
		TotalPosts: optionalInt(initialData["totalPosts"]),
		Next:       parseCursor(initialData),
	}
}

func parsePosts(target map[string]any) ([]*Post, int) {
	rawPosts := slice(target["posts"])
	posts := make([]*Post, 0, len(rawPosts))
	for _, raw := range rawPosts {
		if parsed := ParseItem(obj(raw), ParsePostItem); parsed != nil {
			if post, ok := parsed.(*Post); ok {
				posts = append(posts, post)
			}
		}
	}
	return posts, len(rawPosts)
}

// ParseNoteTimeline parses a note API response.
func ParseNoteTimeline(initialData map[string]any) *NoteTimeline {
	if timeline := obj(initialData["timeline"]); timeline != nil {
		return parseNoteTimelineFrom(timeline, initialData)
	}
	if notes := slice(initialData["notes"]); notes != nil {
		return parseNoteSequence(initialData)
	}
	return nil
}

func parseNoteTimelineFrom(timeline, initialData map[string]any) *NoteTimeline {
	var notes []any
	for _, raw := range slice(timeline["elements"]) {
		if parsed := ParseItem(obj(raw), ParseReplyNoteItem, ParseReblogNoteItem); parsed != nil {
			notes = append(notes, parsed)
		}
	}

	var beforeTimestamp, afterID string
	if queryParams := obj(digDict(timeline, "links", "next", "queryParams")); queryParams != nil {
		beforeTimestamp = str(queryParams["beforeTimestamp"])
		afterID = str(queryParams["after"])
	}

	return noteModel(initialData, notes, beforeTimestamp, afterID)
}

func parseNoteSequence(initialData map[string]any) *NoteTimeline {
	var notes []any
	for _, raw := range slice(initialData["notes"]) {
		if parsed := ParseItem(obj(raw), ParseLikeNoteItem, ParseReblogNoteItem); parsed != nil {
			notes = append(notes, parsed)
		}
	}

	beforeTimestamp := str(digDict(initialData, "links", "next", "queryParams", "beforeTimestamp"))
	return noteModel(initialData, notes, beforeTimestamp, "")
}

func noteModel(initialData map[string]any, notes []any, beforeTimestamp, afterID string) *NoteTimeline {
	return &NoteTimeline{
		Notes:           notes,
		TotalNotes:      integer(initialData["totalNotes"]),
		TotalLikes:      integer(initialData["totalLikes"]),
		TotalReblogs:    integer(initialData["totalReblogs"]),
		TotalReplies:    integer(initialData["totalReplies"]),
		BeforeTimestamp: beforeTimestamp,
		AfterID:         afterID,
	}
}

func parseCursor(initialData map[string]any) *Cursor {
	next := obj(digDict(initialData, "links", "next"))
	if next == nil {
		return nil
	}
	queryParams := obj(next["queryParams"])
	if queryParams == nil {
		return nil
	}

	cursor := str(queryParams["cursor"])
	if cursor == "" {
		cursor = str(queryParams["pageNumber"])
	}

	return &Cursor{
		Cursor:         cursor,
		Limit:          optionalInt(queryParams["days"]),
		Days:           optionalInt(queryParams["query"]),
		Query:          str(queryParams["mode"]),
		Mode:           str(queryParams["timelineType"]),
		SkipComponents: str(queryParams["skipComponent"]),
		ReblogInfo:     optionalBool(queryParams["reblogInfo"]),
		PostTypeFilter: str(queryParams["postTypeFilter"]),
	}
}

func optionalBool(v any) *bool {
	if v == nil {
		return nil
	}
	b := boolean(v)
	return &b
}
