// Package helpers holds hyperblur's shared helpers.
package helpers

import (
	"net/url"
	"strings"
)

// IsTumblrURL reports whether the URL points at tumblr.com or a subdomain.
func IsTumblrURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == "tumblr.com" || strings.HasSuffix(host, ".tumblr.com")
}

// URLHandler rewrites Tumblr URLs found in posts into hyperblur's own
// privacy-friendly paths. It is a port of helpers.url_handler.
func URLHandler(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	hostname := strings.ToLower(u.Hostname())
	path := u.EscapedPath()
	if path == "" {
		path = u.Path
	}

	// Redirect links can hold malformed URLs (e.g. https://href.li/?http://);
	// recurse into the wrapped URL.
	if strings.HasSuffix(hostname, "href.li") {
		return URLHandler(u.RawQuery)
	}
	if strings.HasSuffix(hostname, "t.umblr.com") {
		if values, err := url.ParseQuery(u.RawQuery); err == nil {
			if redirect := values.Get("z"); redirect != "" {
				return URLHandler(redirect)
			}
		}
	}

	if hostname != "" && strings.HasSuffix(hostname, "tumblr.com") {
		if strings.HasSuffix(hostname, ".media.tumblr.com") {
			subDomains := strings.Split(hostname, ".")
			if len(subDomains) > 1 && subDomains[1] == "media" {
				return "/tblr/media/" + subDomains[0] + path
			}
			if len(subDomains) > 2 && subDomains[0] == "www" && subDomains[2] == "media" {
				return "/tblr/media/" + subDomains[1] + path
			}
		}

		switch {
		case strings.HasSuffix(hostname, "assets.tumblr.com"):
			return "/tblr/assets" + path
		case strings.HasSuffix(hostname, "static.tumblr.com"):
			return "/tblr/static" + path
		case strings.HasPrefix(hostname, "a."):
			return "/tblr/a" + path
		default:
			subDomains := strings.Split(hostname, ".")
			potentialBlogName := subDomains[0]
			if subDomains[0] == "www" && len(subDomains) > 1 {
				potentialBlogName = subDomains[1]
			}

			if potentialBlogName != "tumblr" {
				if strings.HasPrefix(path, "/post") {
					return "/" + potentialBlogName + path[5:]
				}
				return "/" + potentialBlogName + path
			}
			return path
		}
	}

	return u.String()
}
