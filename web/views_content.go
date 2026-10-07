package web

import (
	"fmt"
	"strings"

	"github.com/dan1165/openblur/helpers"
	"github.com/dan1165/openblur/i18n"
	"github.com/dan1165/openblur/render"
	"github.com/dan1165/openblur/tumblr"
)

// ---------------------------------------------------------------------------
// Posts
// ---------------------------------------------------------------------------

func (a *App) renderPost(v *view, data *PageData, post *tumblr.Post) {
	v.raw(`<div class="post" data-post-id="`)
	v.esc(post.ID)
	v.raw(`">`)

	postURL := data.PostURL
	if postURL == "" {
		postURL = PostPath(post)
	}

	a.renderPostHeader(v, data, postHeaderData{
		Blog:           post.Blog,
		Date:           post.Date,
		ID:             post.ID,
		Slug:           post.Slug,
		ReblogFrom:     post.ReblogFrom,
		ReblogRoot:     post.ReblogRoot,
		UseThisPostURL: postURL,
	})
	a.renderPostBody(v, data, post)
	a.renderPostFooter(v, data, post, postURL)

	v.raw(`</div>`)
}

func (a *App) renderPostBody(v *view, data *PageData, post *tumblr.Post) {
	hasContent := len(post.Content) > 0

	var mainErr *render.RenderError
	var mainTag string
	if hasContent {
		mainErr, mainTag = a.FormatNPF(post.Content, post.Layout, post.Blog.Name, post.ID, data.RequestPollData, data.ExpandPosts)
	}

	v.raw(`<div class="post-content">`)

	if len(post.Trail) > 0 {
		v.raw(`<div class="post-trails">`)
	}

	for _, trail := range post.Trail {
		v.raw(`<div class="trail-post">`)
		a.renderPostHeader(v, data, postHeaderData{
			Blog:       trail.Blog,
			BrokenBlog: trail.BrokenBlog,
			Date:       trail.Date,
			ID:         trail.ID,
		})
		trailErr, trailTag := a.FormatNPF(trail.Content, trail.Layout, post.Blog.Name, post.ID, data.RequestPollData, data.ExpandPosts)
		if trailErr != nil {
			a.Logger.Printf("npf render error for trail of post %s/%s: %s", post.Blog.Name, post.ID, trailErr.Message)
			a.renderNPFError(v, trailErr)
		}
		v.raw(trailTag)
		v.raw(`</div>`)
	}

	if len(post.Trail) > 0 && hasContent {
		v.raw(`<div class="trail-post">`)
		a.renderPostHeader(v, data, postHeaderData{
			Blog:           post.Blog,
			Date:           post.Date,
			ID:             post.ID,
			Slug:           post.Slug,
			ReblogFrom:     post.ReblogFrom,
			ReblogRoot:     post.ReblogRoot,
			SkipReblog:     true,
			UseThisPostURL: postURLFor(data, post),
		})
		if mainErr != nil {
			a.Logger.Printf("npf render error for post %s/%s: %s", post.Blog.Name, post.ID, mainErr.Message)
			a.renderNPFError(v, mainErr)
		}
		v.raw(mainTag)
		v.raw(`</div>`)
	} else {
		if mainErr != nil {
			a.Logger.Printf("npf render error for post %s/%s: %s", post.Blog.Name, post.ID, mainErr.Message)
			a.renderNPFError(v, mainErr)
		}
		v.raw(mainTag)
	}

	if len(post.Trail) > 0 {
		v.raw(`</div>`)
	}
	v.raw(`</div>`)
}

func postURLFor(data *PageData, post *tumblr.Post) string {
	if data.PostURL != "" {
		return data.PostURL
	}
	return PostPath(post)
}

func (a *App) renderPostFooter(v *view, data *PageData, post *tumblr.Post, postURL string) {
	v.raw(`<footer class="post-footer">`)

	if len(post.Tags) > 0 {
		v.raw(`<div class="post-tags">`)
		for _, tag := range post.Tags {
			if data.BlogTagName != "" {
				v.raw(`<a class="post-tag" href="/`)
				v.esc(data.BlogTagName)
				v.raw(`/tagged/`)
				v.raw(urlEscape(tag))
				v.raw(`"><span>#`)
				v.esc(tag)
				v.raw(`</span></a>`)
			} else {
				v.raw(`<a class="post-tag" href="/tagged/`)
				v.raw(urlEscape(tag))
				v.raw(`"><span>#`)
				v.esc(tag)
				v.raw(`</span></a>`)
			}
		}
		v.raw(`</div>`)
	}

	noteCount := 0
	if post.NoteCount != nil {
		noteCount = *post.NoteCount
	}

	v.raw(`<div class="post-interaction"><div class="note-count"><a href="/`)
	v.esc(postURL)
	v.raw(`?note_viewer=`)
	v.esc(post.DefaultNoteViewerTab)
	v.raw(`"> `)
	v.esc(i18n.Translate("post_note_count", &noteCount, map[string]string{"0": formatDecimal(noteCount)}))
	v.raw(`</a></div><div class="interaction-buttons">`)
	v.raw(`<button type="button" class="copy-link" data-post-url="/`)
	v.esc(postURL)
	v.raw(`" title="`)
	v.esc(a.translate("post_footer_copy_link_icon_title"))
	v.raw(`"><svg xmlns="http://www.w3.org/2000/svg" height="20" viewBox="0 -960 960 960" width="20"><title>`)
	v.esc(a.translate("post_footer_copy_link_icon_title"))
	v.raw(`</title><path d="M440-280H280q-83 0-141.5-58.5T80-480q0-83 58.5-141.5T280-680h160v80H280q-50 0-85 35t-35 85q0 50 35 85t85 35h160v80ZM320-440v-80h320v80H320Zm200 160v-80h160q50 0 85-35t35-85q0-50-35-85t-85-35H520v-80h160q83 0 141.5 58.5T880-480q0 83-58.5 141.5T680-280H520Z"/></svg></button>`)
	v.raw(`<a href="https://www.tumblr.com/`)
	v.esc(postURL)
	v.raw(`" rel="noreferrer"><svg xmlns="http://www.w3.org/2000/svg" height="18px" viewBox="0 0 320 512" role="img" aria-label="`)
	v.esc(a.translate("post_footer_view_on_tumblr_icon_title"))
	v.raw(`"><title>`)
	v.esc(a.translate("post_footer_view_on_tumblr_icon_title"))
	v.raw(`</title><path d="M309.8 480.3c-13.6 14.5-50 31.7-97.4 31.7-120.8 0-147-88.8-147-140.6v-144H17.9c-5.5 0-10-4.5-10-10v-68c0-7.2 4.5-13.6 11.3-16 62-21.8 81.5-76 84.3-117.1.8-11 6.5-16.3 16.1-16.3h70.9c5.5 0 10 4.5 10 10v115.2h83c5.5 0 10 4.4 10 9.9v81.7c0 5.5-4.5 10-10 10h-83.4V360c0 34.2 23.7 53.6 68 35.8 4.8-1.9 9-3.2 12.7-2.2 3.5.9 5.8 3.4 7.4 7.9l22 64.3c1.8 5 3.3 10.6-.4 14.5z"/></svg></a>`)
	v.raw(`</div></div></footer>`)
}

func (a *App) renderNPFError(v *view, renderErr *render.RenderError) {
	v.raw(`<figure class="alert error secondary"><figcaption><svg class="icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 -960 960 960"><path d="M480-280q17 0 28.5-11.5T520-320q0-17-11.5-28.5T480-360q-17 0-28.5 11.5T440-320q0 17 11.5 28.5T480-280Zm-40-160h80v-240h-80v240Zm40 360q-83 0-156-31.5T197-197q-54-54-85.5-127T80-480q0-83 31.5-156T197-763q54-54 127-85.5T480-880q83 0 156 31.5T763-763q54 54 85.5 127T880-480q0 83-31.5 156T763-197q-54 54-127 85.5T480-80Z"/></svg><h4>`)
	v.esc(a.translate("alert_error_on_rendering_post_contents_heading"))
	v.raw(`</h4><p>`)
	v.esc(a.translate("alert_error_on_rendering_post_contents"))
	v.raw(`</p></figcaption>`)
	a.renderErrorDetails(v, renderErr.Name, renderErr.Message, renderErr.Context, false)
	v.raw(`</figure>`)
}

func (a *App) renderErrorDetails(v *view, name, message, context string, open bool) {
	v.raw(`<details`)
	if open {
		v.raw(` open=""`)
	}
	v.raw(` class="error-technical-details"><summary>`)
	v.esc(a.translate("openblur_error_generic_technical_details_expansion_box_label"))
	v.raw(`</summary><pre>Error: `)
	v.esc(name)
	v.raw(`</pre>`)
	if message != "" {
		v.raw(`<pre>Message: "`)
		v.esc(message)
		v.raw(`"</pre>`)
	}
	v.raw(`<pre>Context:<br/><br/>`)
	v.esc(context)
	v.raw(`</pre></details>`)
}

// ---------------------------------------------------------------------------
// Timelines
// ---------------------------------------------------------------------------

func (a *App) renderTimelineCenter(v *view, data *PageData, controlBar func(v *view), paging func(v *view)) {
	v.raw(`<ul id="timeline-control-bar">`)
	if controlBar != nil {
		controlBar(v)
	}
	v.raw(`</ul>`)

	if len(data.Timeline.Signposts) > 0 {
		a.renderAlerts(v, data)
	}

	v.raw(`<div class="timeline" id="m">`)
	for _, element := range data.Timeline.Elements {
		if post, ok := element.(*tumblr.Post); ok && !post.IsAdvertisement {
			a.renderPost(v, data, post)
		}
	}
	v.raw(`</div>`)

	if paging != nil && data.Timeline.Next != "" {
		v.raw(`<div class="paging">`)
		paging(v)
		v.raw(`</div>`)
	}
}

func (a *App) renderAlerts(v *view, data *PageData) {
	v.raw(`<div class="alerts">`)
	for _, signpost := range data.Timeline.Signposts {
		switch signpost.Title {
		case "Hold your horses!":
			v.raw(`<figure class="alert"><figcaption><h4>`)
			v.esc(a.translate("alert_restricted_results_heading"))
			v.raw(`</h4><p>`)
			v.esc(a.translate("alert_restricted_results_message"))
			v.raw(`</p></figcaption></figure>`)
		case "Woah, hang on there.":
			v.raw(`<figure class="alert"><figcaption><h4>`)
			v.esc(a.translate("alert_partial_restricted_results_heading"))
			v.raw(`</h4><p>`)
			v.esc(a.translate("alert_partial_restricted_results_message"))
			v.raw(`</p></figcaption></figure>`)
		default:
			v.raw(`<figure class="tumblr-signpost"><svg class="icon" xmlns="http://www.w3.org/2000/svg" height="48px" width="48px" viewBox="0 -960 960 960"><path d="M451.5-85v-185H234L125-379l109-108.5h217.5v-90H165V-795h286.5v-80H509v80h217.5L835-686.5l-108.5 109H509v90h286V-270H509v185h-57.5Z"/></svg><div><h3>`)
			v.esc(signpost.Title)
			v.raw(`</h3><p>`)
			v.esc(signpost.Description)
			v.raw(`</p></div></figure>`)
		}
	}
	v.raw(`</div>`)
}

// ---------------------------------------------------------------------------
// Blog header
// ---------------------------------------------------------------------------

func (a *App) renderBlogHeader(v *view, data *PageData) {
	blogInfo := data.Blog.BlogInfo

	var descriptionTag string
	if len(blogInfo.DescriptionNPF) > 0 {
		if _, tag := a.FormatNPF(blogInfo.DescriptionNPF, nil, "", "", false, data.ExpandPosts); tag != "" {
			descriptionTag = tag
		}
	}

	banner := helpers.URLHandler(blogInfo.Banner)
	avatar := ""
	if len(blogInfo.Avatar) >= 2 {
		avatar = helpers.URLHandler(blogInfo.Avatar[len(blogInfo.Avatar)-2].URL)
	} else if len(blogInfo.Avatar) == 1 {
		avatar = helpers.URLHandler(blogInfo.Avatar[0].URL)
	}

	v.raw(`<header id="blog-header"><img id="banner" alt="`)
	v.esc(a.translate("blog_banner_alt"))
	v.raw(`" src="`)
	v.esc(banner)
	v.raw(`"/><a href="/`)
	v.esc(blogInfo.Name)
	v.raw(`"><img class="avatar" alt="`)
	v.esc(a.translate("blog_avatar_alt"))
	v.raw(`" src="`)
	v.esc(avatar)
	v.raw(`"/></a><div class="blog-header-textual-content">`)
	if blogInfo.Title != "" {
		v.raw(`<h1 id="blog-title">`)
		v.esc(blogInfo.Title)
		v.raw(`</h1>`)
	}
	v.raw(`<p class="blog-name"><a href="/`)
	v.esc(blogInfo.Name)
	v.raw(`">@`)
	v.esc(blogInfo.Name)
	v.raw(`</a></p><div id="blog-description">`)
	v.raw(descriptionTag)
	v.raw(`</div></div></header>`)
}

// translate is a convenience for handlers.
func (a *App) translate(id string) string { return i18n.Translate(id, nil, nil) }

func urlEscape(s string) string {
	const unreserved = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~"
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if strings.IndexByte(unreserved, c) >= 0 {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}
