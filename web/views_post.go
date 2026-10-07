package web

import (
	"net/url"
	"strings"
	"time"

	"github.com/dan1165/hyperblur/render"
	"github.com/dan1165/hyperblur/tumblr"
)

// Icons reused from the templates.
const (
	reblogIcon   = `<svg class="icon" xmlns="http://www.w3.org/2000/svg" height="16" viewBox="0 -960 960 960" width="16"><path d="M280-80 120-240l160-160 56 58-62 62h406v-160h80v240H274l62 62-56 58Zm-80-440v-240h486l-62-62 56-58 160 160-160 160-56-58 62-62H280v160h-80Z"/></svg>`
	dropdownIcon = `<svg class="icon" xmlns="http://www.w3.org/2000/svg" height="16" viewBox="0 -960 960 960" width="16"><path d="M480-345 240-585l56-56 184 184 184-184 56 56-240 240Z"/></svg>`
	heartIcon    = `<svg class="icon" xmlns="http://www.w3.org/2000/svg" height="24" viewBox="0 -960 960 960" width="24"><path d="M479.62-171.62q-12.85 0-25.81-4.61-12.96-4.62-22.81-14.46l-57.46-52.23q-106.38-97-189.96-190.58Q100-527.08 100-634q0-85.15 57.42-142.58Q214.85-834 300-834q48.38 0 95.58 22.31 47.19 22.31 84.42 72.46 37.23-50.15 84.42-72.46Q611.62-834 660-834q85.15 0 142.58 57.42Q860-719.15 860-634q0 108.08-85 202.73-85 94.65-189.54 188.96l-56.85 51.62q-9.84 9.84-22.99 14.46-13.16 4.61-26 4.61Z"/></svg>`
	replyIcon    = `<svg class="icon" xmlns="http://www.w3.org/2000/svg" height="24" viewBox="0 -960 960 960" width="24"><path d="M80-80v-733.33q0-27 19.83-46.84Q119.67-880 146.67-880h666.66q27 0 46.84 19.83Q880-840.33 880-813.33v506.66q0 27-19.83 46.84Q840.33-240 813.33-240H240L80-80Zm131.33-226.67h602v-506.66H146.67v575l64.66-68.34Zm-64.66 0v-506.66 506.66Z"/></svg>`
)

type postHeaderData struct {
	Blog           *tumblr.Blog
	BrokenBlog     *tumblr.BrokenBlog
	Date           *time.Time
	ReblogFrom     *tumblr.ReblogAttribution
	ReblogRoot     *tumblr.ReblogAttribution
	SkipReblog     bool
	UseThisPostURL string
}

// PostPath returns the post's path (without a leading slash).
func PostPath(post *tumblr.Post) string {
	path := post.Blog.Name + "/" + post.ID
	if post.Slug != "" {
		path += "/" + post.Slug
	}
	return path
}

func (a *App) renderPostHeader(v *view, data *PageData, hd postHeaderData) {
	v.raw(`<div class="post-header"><div class="post-author">`)

	active := hd.Blog != nil && hd.Blog.Active
	if active {
		v.raw(`<a href="/`)
		v.esc(hd.Blog.Name)
		v.raw(`"><img class="avatar" alt="`)
		v.esc("Blog avatar")
		v.raw(`" loading="lazy" src="`)
		v.esc(render.URLHandler(hd.Blog.AvatarURL()))
		v.raw(`"></a>`)
	} else {
		v.raw(`<img class="avatar" alt="`)
		v.esc("Blog avatar")
		v.raw(`" loading="lazy" src="/assets/images/anon_96px.png">`)
	}

	v.raw(`<div class="author-information"><div class="primary-post-author">`)

	switch {
	case hd.Blog != nil && !hd.Blog.Active:
		v.raw(`<span class="link blog-name deactivated-blog">`)
		v.esc(hd.Blog.Name)
		v.raw(`</span>`)
	case hd.Blog != nil:
		v.raw(`<a class="link blog-name" href="/`)
		v.esc(hd.Blog.Name)
		v.raw(`">`)
		v.esc(hd.Blog.Name)
		v.raw(`</a>`)
	default:
		v.raw(`<span class="link blog-name broken-blog" href="/`)
		v.esc(hd.BrokenBlog.Name)
		v.raw(`">`)
		v.esc(hd.BrokenBlog.Name)
		v.raw(`</span>`)
	}

	if hd.Date != nil {
		v.raw(`<span class="separator">•</span><span class="post-timestamp" title="`)
		v.esc(FormatDatetime(*hd.Date))
		v.raw(`"><a href="/`)
		v.esc(hd.UseThisPostURL)
		v.raw(`"><time datetime="`)
		v.esc(hd.Date.Format("2006-01-02T15:04"))
		v.raw(`">`)
		v.esc(FormatDate(*hd.Date))
		v.raw(`</time></a></span>`)
	}

	v.raw(`</div>`)

	if hd.ReblogFrom != nil && !hd.SkipReblog {
		v.raw(`<div class="reblog-attribution">`)
		v.raw(reblogIcon)
		v.raw(a.reblogAttribution(hd.ReblogFrom, hd.ReblogRoot))
		v.raw(`</div>`)
	}

	v.raw(`</div></div></div>`)
}

// reblogAttribution renders who the post was reblogged from.
func (a *App) reblogAttribution(from, root *tumblr.ReblogAttribution) string {
	classes := []string{"link", "blog-name"}
	name := from.BlogName
	if name == "" {
		name = "reblogged"
		classes = append(classes, "hidden-reblog")
	}

	url := from.PostURL
	if !isTumblrURL(url) {
		if root != nil && root.PostID == from.PostID && root.BlogName != "" {
			url = "/" + root.BlogName + "/" + from.PostID
		} else {
			return `<span class="` + strings.Join(classes, " ") + `">` + htmlEscape(name) + `</span>`
		}
	}

	return `<a href="` + htmlEscape(render.URLHandler(url)) + `" class="` + strings.Join(classes, " ") + `">` + htmlEscape(name) + `</a>`
}

func isTumblrURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == "tumblr.com" || strings.HasSuffix(host, ".tumblr.com")
}
