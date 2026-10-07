package web

import (
	"github.com/dan1165/hyperblur/render"
	"github.com/dan1165/hyperblur/tumblr"
)

// renderNotesViewer writes the post-notes section for a post.
func (a *App) renderNotesViewer(v *view, data *PageData) {
	notes := data.Notes
	if notes == nil {
		return
	}

	v.raw(`<section class="post-notes"><header><ul class="post-notes-nav">`)
	a.notesTab(v, data, "replies", replyIcon, "Replies", notes.TotalReplies)
	a.notesTab(v, data, "reblogs", reblogIcon, "Reblogs", notes.TotalReblogs)
	a.notesTab(v, data, "likes", heartIcon, "Likes", notes.TotalLikes)
	v.raw(`</ul><ul id="timeline-control-bar">`)
	a.renderNotesControlBar(v, data)
	v.raw(`</ul></header><section>`)
	if len(notes.Notes) > 0 {
		for _, note := range notes.Notes {
			switch n := note.(type) {
			case *tumblr.ReplyNote:
				a.renderReplyNote(v, data, n)
			case *tumblr.ReblogNote:
				a.renderReblogNote(v, data, n)
			case *tumblr.LikeNote:
				a.renderLikeNote(v, data, n)
			}
		}
	}
	v.raw(`</section><footer>`)
	a.renderNotesPaging(v, data)
	v.raw(`</footer></section>`)
}

func (a *App) notesTab(v *view, data *PageData, noteType, icon, title string, count int) {
	v.raw(`<li`)
	if data.NoteType == noteType {
		v.raw(` class="selected"`)
	}
	v.raw(`><a href="/`)
	v.esc(data.PostURL)
	v.raw(`?note_viewer=`)
	v.raw(noteType)
	v.raw(`" title="`)
	v.esc(title)
	v.raw(`">`)
	v.raw(icon)
	v.raw(`<span>`)
	v.esc(formatDecimal(count))
	v.raw(`</span></a></li>`)
}

func (a *App) renderNotesControlBar(v *view, data *PageData) {
	switch data.NoteType {
	case "replies":
		v.raw(`<li class="control-bar-action no-js" id="sort-by-filter" title="`)
		v.esc("Sort by")
		v.raw(`"><span>`)
		v.esc("Filter")
		v.raw(dropdownIcon)
		v.raw(`</span><ul class="control-bar-dropdown-menu">`)
		v.raw(searchFilterItem(!data.Latest, "/"+data.PostURL+"?note_viewer=replies", "Oldest first"))
		v.raw(searchFilterItem(data.Latest, "/"+data.PostURL+"?note_viewer=replies&latest", "Newest first"))
		v.raw(`</ul></li>`)
	case "reblogs":
		filter := data.ReblogFilter
		if filter == "" {
			filter = "reblogs_with_comments"
		}
		v.raw(`<li class="control-bar-action no-js" id="sort-by-filter" title="`)
		v.esc("Sort by")
		v.raw(`"><span>`)
		v.esc("Filter")
		v.raw(dropdownIcon)
		v.raw(`</span><ul class="control-bar-dropdown-menu">`)
		v.raw(searchFilterItem(filter == "reblogs_with_comments", "/"+data.PostURL+"?note_viewer=reblogs", "Comments and tags"))
		v.raw(searchFilterItem(filter == "reblogs_with_content_comments", "/"+data.PostURL+"?note_viewer=reblogs&reblog_filter=reblogs_with_content_comments", "Comments only"))
		v.raw(searchFilterItem(filter == "reblogs_only", "/"+data.PostURL+"?note_viewer=reblogs&reblog_filter=reblogs_only", "Other reblogs"))
		v.raw(`</ul></li>`)
	}
}

func (a *App) renderNotesPaging(v *view, data *PageData) {
	notes := data.Notes
	if notes == nil {
		return
	}
	switch data.NoteType {
	case "replies":
		if notes.AfterID != "" {
			v.raw(`<a class="secondary next-page button" href="/`)
			v.esc(data.PostURL)
			v.raw(`?`)
			v.raw(htmlEscape(updateQuery(data.QueryArgs, "after", notes.AfterID)))
			v.raw(`">`)
			v.esc("Next page")
			v.raw(`</a>`)
		}
	default:
		if notes.BeforeTimestamp != "" {
			v.raw(`<a class="secondary next-page button" href="/`)
			v.esc(data.PostURL)
			v.raw(`?`)
			v.raw(htmlEscape(updateQuery(data.QueryArgs, "before_timestamp", notes.BeforeTimestamp)))
			v.raw(`">`)
			v.esc("Next page")
			v.raw(`</a>`)
		}
	}
}

func (a *App) renderReplyNote(v *view, data *PageData, note *tumblr.ReplyNote) {
	blog := note.Blog

	v.raw(`<div class="note reply" data-id="`)
	v.esc(note.ReplyID)
	v.raw(`">`)

	renderAvatar(v, blog)

	v.raw(`<div><div class="author-information"><div class="primary-post-author">`)
	renderBlogName(v, blog, nil)
	renderPostDate(v, note.Date, "")
	v.raw(`</div></div><p>`)

	blogName := ""
	if blog != nil {
		blogName = blog.Name
	}
	_, tag := a.FormatNPF(note.Content, note.Layout, blogName, "", data.RequestPollData)
	v.raw(tag)
	v.raw(`</p></div></div>`)
}

func (a *App) renderReblogNote(v *view, data *PageData, note *tumblr.ReblogNote) {
	// The template only ever renders the "blog name absent" branch, since
	// reblog notes never expose a top-level blog_name.
	v.raw(`<div class="note reblog-note post" data-post-id="`)
	v.esc(note.ID)
	v.raw(`">`)
	postURL := ""
	if note.Blog != nil {
		postURL = note.Blog.Name + "/" + note.ID
	}
	a.renderPostHeader(v, data, postHeaderData{
		Blog:           note.Blog,
		Date:           note.Date,
		UseThisPostURL: postURL,
	})
	v.raw(`<div class="post-content">`)
	_, tag := a.FormatNPF(note.Content, note.Layout, note.Blog.Name, note.ID, data.RequestPollData)
	v.raw(tag)
	v.raw(`</div>`)
	if len(note.Tags) > 0 {
		v.raw(`<footer class="post-footer">`)
		renderTags(v, note.Tags, "")
		v.raw(`</footer>`)
	}
	v.raw(`</div><hr>`)
}

func (a *App) renderLikeNote(v *view, _ *PageData, note *tumblr.LikeNote) {
	avatar := ""
	if note.Avatar != nil {
		avatar = note.Avatar["128"]
	}
	v.raw(`<div class="note like"><div class="post-author">`)
	renderAvatarLink(v, note.BlogName, render.URLHandler(avatar))
	v.raw(`<div class="author-information"><div class="primary-post-author"><div class="blog-name-title-grouping"><a class="link blog-name" href="/`)
	v.esc(note.BlogName)
	v.raw(`">`)
	v.esc(note.BlogName)
	v.raw(`</a><span>`)
	v.esc(note.BlogTitle)
	v.raw(`</span></div></div></div></div></div>`)
}
