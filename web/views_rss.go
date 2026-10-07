package web

import (
	"time"

	"github.com/dan1165/openblur/helpers"
	"github.com/dan1165/openblur/tumblr"
)

// renderTimelineRSS writes an RSS feed for a timeline page.
func (a *App) renderTimelineRSS(data *PageData) string {
	title := data.Title
	if data.Query != "" {
		title = data.Query
	} else if data.Tag != "" {
		title = "#" + data.Tag
	}

	data.Updated = lastPostTime(data.Timeline.Elements, data.Updated)

	v := &view{}
	a.rssChannelOpen(v, title, title, data)

	for _, element := range data.Timeline.Elements {
		if post, ok := element.(*tumblr.Post); ok && !post.IsAdvertisement {
			a.rssPostItem(v, data, post)
		}
	}
	a.rssChannelClose(v)
	return v.string()
}

// renderBlogRSS writes an RSS feed for a blog page.
func (a *App) renderBlogRSS(data *PageData) string {
	blogInfo := data.Blog.BlogInfo

	description := ""
	if len(blogInfo.DescriptionNPF) > 0 {
		if _, tag := a.FormatNPF(blogInfo.DescriptionNPF, nil, "", "", false, data.ExpandPosts); tag != "" {
			description = tag
		}
	}

	v := &view{}
	v.raw(`<?xml version="1.0" encoding="utf-8"?>` + "\n")
	v.raw(`<rss xmlns:atom="http://www.w3.org/2005/Atom" xmlns:media="http://search.yahoo.com/mrss/" version="2.0">` + "\n")
	v.raw(`  <channel>` + "\n")
	rssElem(v, "title", blogInfo.Title+" "+a.translate("page_title_suffix"))
	rssElem(v, "description", description)
	rssElem(v, "link", data.PageURL)
	v.raw(`    <atom:link href="` + htmlEscape(data.PageURL) + `/rss" rel="self" type="application/rss+xml" />` + "\n")
	data.Updated = lastPostTime(postsAsAny(data.Blog.Posts), data.Updated)
	a.rssUpdated(v, data)
	avatar := blogAvatarURL(blogInfo)
	v.raw(`    <image><url>` + htmlEscape(helpers.URLHandler(avatar)) + `</url><title>` + htmlEscape(a.translate("blog_avatar_alt")) + `</title><link>` + htmlEscape(data.PageURL) + `</link></image>` + "\n")
	v.raw(`    <atom:logo><atom:icon>` + htmlEscape(helpers.URLHandler(avatar)) + `</atom:icon></atom:logo>` + "\n")
	v.raw(`    <atom:author><name>` + htmlEscape(blogInfo.Name) + `</name></atom:author>` + "\n")

	for _, post := range data.Blog.Posts {
		a.rssPostItem(v, data, post)
	}

	v.raw(`  </channel>` + "\n")
	v.raw(`</rss>` + "\n")
	return v.string()
}

// renderBlogPostRSS writes a single-item RSS feed for a post.
func (a *App) renderBlogPostRSS(data *PageData) string {
	blogInfo := data.Post.Blog

	description := ""
	if len(blogInfo.DescriptionNPF) > 0 {
		if _, tag := a.FormatNPF(blogInfo.DescriptionNPF, nil, "", "", false, data.ExpandPosts); tag != "" {
			description = tag
		}
	}

	v := &view{}
	v.raw(`<?xml version="1.0" encoding="utf-8"?>` + "\n")
	v.raw(`<rss xmlns:atom="http://www.w3.org/2005/Atom" xmlns:media="http://search.yahoo.com/mrss/" version="2.0">` + "\n")
	v.raw(`  <channel>` + "\n")
	rssElem(v, "title", blogInfo.Title+" "+a.translate("page_title_suffix"))
	rssElem(v, "description", description)
	rssElem(v, "link", data.PageURL)
	v.raw(`    <atom:link href="` + htmlEscape(data.PostURL) + `rss_feed=1" rel="self" type="application/rss+xml" />` + "\n")
	if data.Post.Date != nil {
		data.Updated = *data.Post.Date
	}
	a.rssUpdated(v, data)
	avatar := blogAvatarURL(blogInfo)
	v.raw(`    <image><url>` + htmlEscape(helpers.URLHandler(avatar)) + `</url><title>` + htmlEscape(a.translate("blog_avatar_alt")) + `</title><link>` + htmlEscape(data.PageURL) + `</link></image>` + "\n")
	v.raw(`    <atom:logo><atom:icon>` + htmlEscape(helpers.URLHandler(avatar)) + `</atom:icon></atom:logo>` + "\n")
	v.raw(`    <atom:author><name>` + htmlEscape(blogInfo.Name) + `</name></atom:author>` + "\n")

	a.rssPostItem(v, data, data.Post)

	v.raw(`  </channel>` + "\n")
	v.raw(`</rss>` + "\n")
	return v.string()
}

func (a *App) rssChannelOpen(v *view, title, description string, data *PageData) {
	v.raw(`<?xml version="1.0" encoding="utf-8"?>` + "\n")
	v.raw(`<rss xmlns:atom="http://www.w3.org/2005/Atom" xmlns:media="http://search.yahoo.com/mrss/" version="2.0">` + "\n")
	v.raw(`  <channel>` + "\n")
	rssElem(v, "title", title+" "+a.translate("page_title_suffix"))
	rssElem(v, "description", description)
	rssElem(v, "link", data.PageURL)
	v.raw(`    <atom:link href="` + htmlEscape(data.PageURL) + `/rss" rel="self" type="application/rss+xml" />` + "\n")
	a.rssUpdated(v, data)
}

func (a *App) rssChannelClose(v *view) {
	v.raw(`  </channel>` + "\n")
	v.raw(`</rss>` + "\n")
}

func (a *App) rssUpdated(v *view, data *PageData) {
	updated := data.Updated
	v.raw(`    <atom:updated>` + htmlEscape(updated.Format("2006-01-02T15:04:05-07:00")) + `</atom:updated>` + "\n")
	v.raw(`    <lastBuildDate>` + htmlEscape(updated.Format("Mon, 02 Jan 2006 15:04:05 -0700")) + `</lastBuildDate>` + "\n")
}

func (a *App) rssPostItem(v *view, data *PageData, post *tumblr.Post) {
	postURL := "/" + PostPath(post)

	title := post.Summary
	if title == "" {
		title = post.Blog.Title
	}
	if title == "" {
		title = post.Blog.Name
	}

	v.raw(`    <item>` + "\n")
	rssElem(v, "title", title)
	rssElem(v, "link", postURL)
	v.raw(`      <atom:link rel="self" href="` + htmlEscape(postURL) + `?rss_feed=1" type="application/rss+xml"/>` + "\n")
	v.raw(`      <atom:link rel="alternate" href="` + htmlEscape(postURL) + `"/>` + "\n")

	body := &view{}
	a.renderPostBody(body, data, post)
	v.raw(`      <description>` + htmlEscape(body.string()) + `</description>` + "\n")

	v.raw(`      <guid isPermaLink="false">` + htmlEscape(post.ID) + `</guid>` + "\n")
	if post.Date != nil {
		v.raw(`      <pubDate>` + htmlEscape(post.Date.Format("Mon, 02 Jan 2006 15:04:05 -0700")) + `</pubDate>` + "\n")
	}

	for _, trail := range post.Trail {
		v.raw(`      <atom:author><atom:name>` + htmlEscape(trail.BlogName()) + `</atom:name></atom:author>` + "\n")
	}
	if len(post.Content) > 0 {
		v.raw(`      <atom:author><atom:name>` + htmlEscape(post.Blog.Name) + `</atom:name></atom:author>` + "\n")
	}
	for _, tag := range post.Tags {
		rssElem(v, "category", tag)
	}
	v.raw(`      <comments>` + htmlEscape(postURL) + `?note_viewer=` + htmlEscape(post.DefaultNoteViewerTab) + `</comments>` + "\n")
	v.raw(`    </item>` + "\n")
}

func rssElem(v *view, tag, value string) {
	v.raw("    <" + tag + ">" + htmlEscape(value) + "</" + tag + ">\n")
}

func blogAvatarURL(blogInfo *tumblr.Blog) string {
	if len(blogInfo.Avatar) >= 2 {
		return blogInfo.Avatar[len(blogInfo.Avatar)-2].URL
	}
	if len(blogInfo.Avatar) == 1 {
		return blogInfo.Avatar[0].URL
	}
	return ""
}

func postsAsAny(posts []*tumblr.Post) []any {
	out := make([]any, len(posts))
	for i, post := range posts {
		out[i] = post
	}
	return out
}

func lastPostTime(elements []any, fallback time.Time) time.Time {
	for i := len(elements) - 1; i >= 0; i-- {
		if post, ok := elements[i].(*tumblr.Post); ok && post.Date != nil {
			return *post.Date
		}
	}
	return fallback
}
