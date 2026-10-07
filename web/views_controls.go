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
	b.WriteString(htmlEscape(a.translate("timeline_search_sort_by_filter_title")))
	b.WriteString(`"><span>`)
	b.WriteString(htmlEscape(a.translate("timeline_search_sort_by_filter_" + data.SortBy)))
	b.WriteString(dropdownIcon)
	b.WriteString(`</span><ul class="control-bar-dropdown-menu">`)
	if data.PostFilter != "" {
		filter := urlEscape(data.PostFilter)
		b.WriteString(searchSortItem(data.SortBy == "popular", "/search/"+query+"/"+filter+addQuery(deseq(base)), a.translate("timeline_search_sort_by_filter_popular")))
		b.WriteString(searchSortItem(data.SortBy == "recent", "/search/"+query+"/recent/"+filter+addQuery(deseq(base)), a.translate("timeline_search_sort_by_filter_recent")))
	} else {
		b.WriteString(searchSortItem(data.SortBy == "popular", "/search/"+query+addQuery(deseq(base)), a.translate("timeline_search_sort_by_filter_popular")))
		b.WriteString(searchSortItem(data.SortBy == "recent", "/search/"+query+"/recent"+addQuery(deseq(base)), a.translate("timeline_search_sort_by_filter_recent")))
	}
	b.WriteString(`</ul></li>`)

	// Date filter (popular only).
	if data.SortBy == "popular" {
		b.WriteString(`<li class="control-bar-action no-js" id="filter-by-date-filter" title="`)
		b.WriteString(htmlEscape(a.translate("timeline_search_filter_by_date_filter_title")))
		b.WriteString(`"><span>`)
		b.WriteString(htmlEscape(a.translate("timeline_search_filter_by_date_filter_" + data.TimeFilter)))
		b.WriteString(dropdownIcon)
		b.WriteString(`</span><ul class="control-bar-dropdown-menu">`)
		b.WriteString(searchFilterItem(data.TimeFilter == "0", htmlEscape(data.Path)+addQuery(removeQuery(base, "t")), a.translate("timeline_search_filter_by_date_filter_0")))
		for _, time := range []string{"365", "180", "30", "7", "1"} {
			b.WriteString(searchFilterItem(data.TimeFilter == time, htmlEscape(data.Path)+addQuery(updateQuery(base, "t", time)), a.translate("timeline_search_filter_by_date_filter_"+time)))
		}
		b.WriteString(`</ul></li>`)
	}

	// Post type filter.
	selectedLabel := a.translate("timeline_search_post_type_filter_none")
	if data.PostFilter != "" {
		selectedLabel = a.translate("timeline_search_post_type_filter_" + data.PostFilter)
	}
	b.WriteString(`<li class="control-bar-action no-js" id="filter-by-post-type-filter" title="`)
	b.WriteString(htmlEscape(a.translate("timeline_search_post_type_filter_title")))
	b.WriteString(`"><span>`)
	b.WriteString(htmlEscape(selectedLabel))
	b.WriteString(dropdownIcon)
	b.WriteString(`</span><ul class="control-bar-dropdown-menu">`)

	if data.SortBy == "recent" {
		b.WriteString(searchFilterItem(data.PostFilter == "", "/search/"+query+"/recent"+addQuery(deseq(base)), a.translate("timeline_search_post_type_filter_none")))
	} else {
		b.WriteString(searchFilterItem(data.PostFilter == "", "/search/"+query+addQuery(deseq(base)), a.translate("timeline_search_post_type_filter_none")))
	}
	for _, postType := range []string{"text", "photo", "gif", "quote", "link", "chat", "audio", "video", "ask", "poll"} {
		target := "/search/" + query + "/" + urlEscape(postType) + addQuery(deseq(base))
		if data.SortBy == "recent" {
			target = "/search/" + query + "/recent/" + urlEscape(postType) + addQuery(deseq(base))
		}
		b.WriteString(searchFilterItem(data.PostFilter == postType, target, a.translate("timeline_search_post_type_filter_"+postType)))
	}
	b.WriteString(`</ul></li>`)

	v.raw(b.String())
}

func searchSortItem(selected bool, href, label string) string {
	class := ""
	if selected {
		class = ` class="selected"`
	}
	return `<li` + class + `><a href="` + href + `">` + htmlEscape(label) + `</a></li>`
}

func searchFilterItem(selected bool, href, label string) string {
	class := ""
	if selected {
		class = ` class="selected"`
	}
	return `<li` + class + `><a href="` + href + `">` + htmlEscape(label) + `</a></li>`
}

func (a *App) renderTaggedControlBar(v *view, data *PageData) {
	base := data.QueryArgs
	tag := urlEscape(data.Tag)

	v.raw(`<li class="control-bar-action no-js" id="sort-by-filter" title="`)
	v.esc(a.translate("timeline_tagged_sort_by_filter_title"))
	v.raw(`"><span>`)
	v.esc(a.translate("timeline_tagged_sort_by_filter_" + data.SortBy))
	v.raw(dropdownIcon)
	v.raw(`</span><ul class="control-bar-dropdown-menu">`)
	v.raw(searchFilterItem(data.SortBy == "top", "/tagged/"+tag+addQuery(updateQuery(base, "sort", "top")), a.translate("timeline_tagged_sort_by_filter_top")))
	v.raw(searchFilterItem(data.SortBy == "recent", "/tagged/"+tag+addQuery(updateQuery(base, "sort", "recent")), a.translate("timeline_tagged_sort_by_filter_recent")))
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
	v.esc(a.translate("pagination_next_page"))
	v.raw(`</a>`)
}
