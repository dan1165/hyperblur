package web

import (
	"net/url"
	"strings"
)

func deseq(values url.Values) string { return values.Encode() }

func addQuery(query string) string {
	if query != "" {
		return "?" + query
	}
	return ""
}

func updateQuery(values url.Values, key, value string) string {
	copyValues := url.Values{}
	for k, v := range values {
		copyValues[k] = append([]string(nil), v...)
	}
	copyValues.Set(key, value)
	return copyValues.Encode()
}

func removeQuery(values url.Values, key string) string {
	copyValues := url.Values{}
	for k, v := range values {
		copyValues[k] = append([]string(nil), v...)
	}
	copyValues.Del(key)
	return copyValues.Encode()
}

func (a *App) renderSearchControlBar(v *view, data *PageData) {
	query := urlEscape(data.Query)
	base := data.QueryArgs
	var b strings.Builder

	// Sort-by filter.
	b.WriteString(`<li class="control-bar-action no-js" id="sort-by-filter" title="`)
	b.WriteString(htmlEscape("Sort by"))
	b.WriteString(`"><span>`)
	b.WriteString(htmlEscape(sortLabel(data.SortBy, "Popular")))
	b.WriteString(dropdownIcon)
	b.WriteString(`</span><ul class="control-bar-dropdown-menu">`)
	if data.PostFilter != "" {
		filter := urlEscape(data.PostFilter)
		b.WriteString(searchFilterItem(data.SortBy == "popular", "/search/"+query+"/"+filter+addQuery(deseq(base)), "Popular"))
		b.WriteString(searchFilterItem(data.SortBy == "recent", "/search/"+query+"/recent/"+filter+addQuery(deseq(base)), "Latest"))
	} else {
		b.WriteString(searchFilterItem(data.SortBy == "popular", "/search/"+query+addQuery(deseq(base)), "Popular"))
		b.WriteString(searchFilterItem(data.SortBy == "recent", "/search/"+query+"/recent"+addQuery(deseq(base)), "Latest"))
	}
	b.WriteString(`</ul></li>`)

	// Date filter (popular only).
	if data.SortBy == "popular" {
		b.WriteString(`<li class="control-bar-action no-js" id="filter-by-date-filter" title="`)
		b.WriteString(htmlEscape("Filter by date"))
		b.WriteString(`"><span>`)
		b.WriteString(htmlEscape(dateFilterLabel(data.TimeFilter)))
		b.WriteString(dropdownIcon)
		b.WriteString(`</span><ul class="control-bar-dropdown-menu">`)
		b.WriteString(searchFilterItem(data.TimeFilter == "0", htmlEscape(data.Path)+addQuery(removeQuery(base, "t")), "All Time"))
		for _, time := range []string{"365", "180", "30", "7", "1"} {
			b.WriteString(searchFilterItem(data.TimeFilter == time, htmlEscape(data.Path)+addQuery(updateQuery(base, "t", time)), dateFilterLabel(time)))
		}
		b.WriteString(`</ul></li>`)
	}

	// Post type filter.
	selectedLabel := "All Types"
	if data.PostFilter != "" {
		selectedLabel = postTypeLabel(data.PostFilter)
	}
	b.WriteString(`<li class="control-bar-action no-js" id="filter-by-post-type-filter" title="`)
	b.WriteString(htmlEscape("Filter by post type"))
	b.WriteString(`"><span>`)
	b.WriteString(htmlEscape(selectedLabel))
	b.WriteString(dropdownIcon)
	b.WriteString(`</span><ul class="control-bar-dropdown-menu">`)

	if data.SortBy == "recent" {
		b.WriteString(searchFilterItem(data.PostFilter == "", "/search/"+query+"/recent"+addQuery(deseq(base)), "All Types"))
	} else {
		b.WriteString(searchFilterItem(data.PostFilter == "", "/search/"+query+addQuery(deseq(base)), "All Types"))
	}
	for _, postType := range []string{"text", "photo", "gif", "quote", "link", "chat", "audio", "video", "ask", "poll"} {
		target := "/search/" + query + "/" + urlEscape(postType) + addQuery(deseq(base))
		if data.SortBy == "recent" {
			target = "/search/" + query + "/recent/" + urlEscape(postType) + addQuery(deseq(base))
		}
		b.WriteString(searchFilterItem(data.PostFilter == postType, target, postTypeLabel(postType)))
	}
	b.WriteString(`</ul></li>`)

	v.raw(b.String())
}

func searchFilterItem(selected bool, href, label string) string {
	class := ""
	if selected {
		class = ` class="selected"`
	}
	return `<li` + class + `><a href="` + href + `">` + htmlEscape(label) + `</a></li>`
}

// sortLabel returns "Latest" for recency sorting, otherwise topLabel.
func sortLabel(sortBy, topLabel string) string {
	if sortBy == "recent" {
		return "Latest"
	}
	return topLabel
}

func dateFilterLabel(filter string) string {
	switch filter {
	case "1":
		return "Today"
	case "7":
		return "Last week"
	case "30":
		return "Last month"
	case "180":
		return "Last 6 months"
	case "365":
		return "Last year"
	}
	return "All Time"
}

func postTypeLabel(postType string) string {
	switch postType {
	case "text":
		return "Text"
	case "photo":
		return "Photo"
	case "gif":
		return "Gifs"
	case "quote":
		return "Quote"
	case "link":
		return "Link"
	case "chat":
		return "Chat"
	case "audio":
		return "Audio"
	case "video":
		return "Video"
	case "ask":
		return "Ask"
	case "poll":
		return "Poll"
	}
	return "All Types"
}

func (a *App) renderTaggedControlBar(v *view, data *PageData) {
	base := data.QueryArgs
	tag := urlEscape(data.Tag)

	v.raw(`<li class="control-bar-action no-js" id="sort-by-filter" title="`)
	v.esc("Sort by")
	v.raw(`"><span>`)
	v.esc(sortLabel(data.SortBy, "Top"))
	v.raw(dropdownIcon)
	v.raw(`</span><ul class="control-bar-dropdown-menu">`)
	v.raw(searchFilterItem(data.SortBy == "top", "/tagged/"+tag+addQuery(updateQuery(base, "sort", "top")), "Top"))
	v.raw(searchFilterItem(data.SortBy == "recent", "/tagged/"+tag+addQuery(updateQuery(base, "sort", "recent")), "Latest"))
	v.raw(`</ul></li>`)
}

func (a *App) renderSearchPaging(v *view, data *PageData) {
	if data.Timeline == nil || data.Timeline.Next == "" {
		return
	}
	href := htmlEscape(data.Path) + addQuery(updateQuery(data.QueryArgs, "continuation", data.Timeline.Next))
	v.raw(`<a class="primary next-page button" href="`)
	v.raw(href)
	v.raw(`#m">`)
	v.esc("Next page")
	v.raw(`</a>`)
}
