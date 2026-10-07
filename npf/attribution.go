package npf

import (
	"fmt"
	"net/url"
)

func hostname(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

func formatLinkAttribution(attr Attribution, urlHandler func(string) string) *Node {
	return El("div", "class", "link-attribution").
		Add(El("a", "href", urlHandler(attr.URL)).Add(Txt(hostname(attr.URL))))
}

func formatPostAttribution(attr Attribution, urlHandler func(string) string, loc Localizer) *Node {
	name := El("b").Add(Txt(attr.Blog.Name)).Render()
	inner := loc.Translate("post_attribution", map[string]string{"author": name})
	return El("div", "class", "post-attribution").
		Add(El("a", "href", urlHandler(attr.URL)).Add(RawHTML(inner)))
}

func formatBlogAttribution(attr Attribution, urlHandler func(string) string, loc Localizer) *Node {
	name := attr.Blog.Name
	if name == "" {
		name = "Anonymous"
	}
	author := El("b").Add(Txt(name)).Render()
	inner := loc.Translate("blog_attribution", map[string]string{"author": author})
	return El("div", "class", "blog-attribution").
		Add(El("a", "href", urlHandler(attr.URL)).Add(RawHTML(inner)))
}

func formatAppAttribution(attr Attribution, urlHandler func(string) string, loc Localizer) *Node {
	app := El("b").Add(Txt(attr.AppName)).Render()
	inner := loc.Translate("app_attribution", map[string]string{"platform": app})
	return El("div", "class", "post-attribution").
		Add(El("a", "href", urlHandler(attr.URL)).Add(RawHTML(inner)))
}

func formatUnsupportedAttribution(attr Attribution, loc Localizer) *Node {
	inner := loc.Translate("unsupported_attribution", map[string]string{"attributee": attr.TypeStr})
	return El("div", "class", "unknown-attribution").Add(El("p").Add(Txt(inner)))
}

// formatAsk renders an "ask" layout with the given pre-rendered contents.
func formatAsk(attr *BlogAttribution, urlHandler func(string) string, loc Localizer, contents ...*Node) *Node {
	var askerAttribution *Node
	var askerAvatar *Node

	if attr == nil {
		name := El("strong", "class", "asker-name").Add(Txt(loc.Translate("asker_with_no_attribution", nil)))
		asked := loc.Translate("asker_and_ask_verb", map[string]string{"name": name.Render()})
		askerAttribution = El("p", "class", "asker").Add(RawHTML(asked))
		askerAvatar = El("img",
			"src", urlHandler("https://assets.tumblr.com/images/anonymous_avatar_96.gif"),
			"loading", "lazy",
			"class", "avatar asker-avatar image")
	} else {
		nameHTML := El("a",
			"href", urlHandler(fmt.Sprintf("https://%s.tumblr.com/", attr.Name)),
			"class", "asker-attribution").
			Add(El("strong", "class", "asker-name").Add(Txt(attr.Name))).Render()
		asked := loc.Translate("asker_and_ask_verb", map[string]string{"name": nameHTML})
		askerAttribution = El("p", "class", "asker").Add(RawHTML(asked))

		if len(attr.Avatar) > 0 {
			askerAvatar = El("img",
				"src", urlHandler(attr.Avatar[0].URL),
				"loading", "lazy",
				"class", "avatar asker-avatar image")
		}
	}

	ask := El("div", "class", "ask")
	body := El("div", "class", "ask-body")
	header := El("div", "class", "ask-header")
	header.Add(askerAttribution)
	content := El("div", "class", "ask-content")
	content.Add(contents...)
	body.Add(header, content)
	ask.Add(body)
	if askerAvatar != nil {
		ask.Add(askerAvatar)
	}
	return ask
}
