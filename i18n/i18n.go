// Package i18n holds openblur's English UI strings.
package i18n

import "strings"

// Strings maps message ids to their English text.
var Strings = map[string]string{
	"project_title":                                                      "openblur",
	"page_title_suffix":                                                  "- openblur",
	"search_bar_placeholder_text":                                        "Search",
	"navbar_today_on_tumblr_icon_title":                                  "Today on Tumblr",
	"navbar_trending_icon_title":                                         "Trending",
	"dropdown_filter_menu_text":                                          "Filter",
	"timeline_search_sort_by_filter_title":                               "Sort by",
	"timeline_search_sort_by_filter_popular":                             "Popular",
	"timeline_search_sort_by_filter_recent":                              "Latest",
	"timeline_search_filter_by_date_filter_title":                        "Filter by date",
	"timeline_search_filter_by_date_filter_0":                            "All Time",
	"timeline_search_filter_by_date_filter_1":                            "Today",
	"timeline_search_filter_by_date_filter_7":                            "Last week",
	"timeline_search_filter_by_date_filter_30":                           "Last month",
	"timeline_search_filter_by_date_filter_180":                          "Last 6 months",
	"timeline_search_filter_by_date_filter_365":                          "Last year",
	"timeline_search_post_type_filter_none":                              "All Types",
	"timeline_search_post_type_filter_title":                             "Filter by post type",
	"timeline_search_post_type_filter_text":                              "Text",
	"timeline_search_post_type_filter_photo":                             "Photo",
	"timeline_search_post_type_filter_gif":                               "Gifs",
	"timeline_search_post_type_filter_quote":                             "Quote",
	"timeline_search_post_type_filter_link":                              "Link",
	"timeline_search_post_type_filter_chat":                              "Chat",
	"timeline_search_post_type_filter_audio":                             "Audio",
	"timeline_search_post_type_filter_video":                             "Video",
	"timeline_search_post_type_filter_ask":                               "Ask",
	"timeline_search_post_type_filter_poll":                              "Poll",
	"timeline_tagged_sort_by_filter_title":                               "Sort by",
	"timeline_tagged_sort_by_filter_top":                                 "Top",
	"timeline_tagged_sort_by_filter_recent":                              "Latest",
	"explore_trending_page_title":                                        "Trending topics",
	"explore_today_on_tumblr_page_title":                                 "Today on Tumblr",
	"pagination_next_page":                                               "Next page",
	"tumblr_error_blog_login_required_error_heading":                     "This blog requires an account to view",
	"tumblr_error_blog_login_required_error_description":                 "Try finding reblogs instead!",
	"tumblr_error_restricted_tag_error_heading":                          "This tag has been restricted on Tumblr",
	"tumblr_error_restricted_tag_description":                            "Try performing a search instead",
	"tumblr_error_blog_not_found_error_heading":                          "Unable to find the requested blog",
	"tumblr_error_blog_not_found_error_description":                      "The blog may have been deleted or just never existed in the first place",
	"tumblr_error_blog_requires_password_error_heading":                  "This blog requires a password to access",
	"tumblr_error_ratelimit_reached_heading":                             "openblur has been ratelimited by Tumblr",
	"tumblr_error_ratelimit_reached_description":                         "Please try again later",
	"openblur_error_page_title":                                          "Error",
	"openblur_error_request_to_tumblr_timed_out_heading":                 "Error: Request to Tumblr timed out",
	"openblur_error_request_to_tumblr_timed_out_description":             "openblur was unable to complete the request to Tumblr before timing out",
	"openblur_error_invalid_internal_tumblr_redirect":                    "Error: Tumblr HTTP 301 redirect points to foreign URL",
	"openblur_error_generic":                                             "An unknown exception has occured!",
	"openblur_error_generic_description":                                 "It looks like you have found a bug in openblur.",
	"openblur_error_generic_description_2":                               "Please report it here at GitHub",
	"openblur_error_generic_technical_details_expansion_box_label":       "Show error log",
	"post_community_label_mature_heading":                                "Community Label: Mature",
	"post_community_label_generic_explanation":                           "This post may contain content that is not suitable for all audiences.",
	"post_community_label_sexual_themes":                                 "Sexual themes",
	"post_community_label_drug_use":                                      "Drug and alcohol addiction",
	"post_community_label_violence":                                      "Violence",
	"post_community_label_show_post_button":                              "Show post",
	"post_community_label_no_js_show_post_instructions":                  "Reveals on hover",
	"post_footer_copy_link_icon_title":                                   "Copy link",
	"post_footer_view_on_tumblr_icon_title":                              "View on Tumblr",
	"blog_search_placeholder_text":                                       "Search posts",
	"blog_banner_alt":                                                    "Blog banner",
	"blog_avatar_alt":                                                    "Blog avatar",
	"settings_header":                                                    "Settings",
	"settings_language_selector":                                         "Language",
	"settings_language_selector_desc":                                    "Select which language you'd like openblur to use",
	"settings_theme_selector":                                            "Theme",
	"settings_theme_selector_desc":                                       "Select your display theme",
	"settings_theme_selector_option_auto":                                "Auto",
	"settings_theme_selector_option_light":                               "Light",
	"settings_theme_selector_option_dark":                                "Dark",
	"settings_save_changes":                                              "Save Changes",
	"settings_cancel_changes":                                            "Cancel",
	"settings_copy_as_bookmarklet":                                       "Copy as bookmarklet",
	"settings_copy_as_bookmarklet_confirmed":                             "Copied",
	"settings_copy_as_bookmarklet_failed":                                "Unable to copy",
	"post_note_viewer_view_replies_tab_title":                            "Replies",
	"post_note_viewer_view_reblogs_tab_title":                            "Reblogs",
	"post_note_viewer_view_likes_tab_title":                              "Likes",
	"post_note_viewer_view_reblogs_filter_reblogs_with_comments":         "Comments and tags",
	"post_note_viewer_view_reblogs_filter_reblogs_with_content_comments": "Comments only",
	"post_note_viewer_view_reblogs_filter_reblogs_only":                  "Other reblogs",
	"post_note_viewer_view_replies_filter_sort_oldest":                   "Oldest first",
	"post_note_viewer_view_replies_filter_sort_newest":                   "Newest first",
	"settings_expand_blogger_truncated_posts":                            "Expand posts",
	"settings_expand_blogger_truncated_posts_desc":                       "Expands truncated posts automatically",

	"npf_renderer_asker_with_no_attribution": "Anonymous",
	"npf_renderer_asker_and_ask_verb":        "{name} asked",
	"npf_renderer_unsupported_block_header":  "Unsupported NPF block",
	"npf_renderer_unsupported_block_description": "Placeholder for the unsupported \"{block}\" type NPF " +
		"block Please report me over at https://github.com/syeopite/npf-renderer",
	"npf_renderer_generic_image_alt_text":     "image",
	"npf_renderer_link_block_poster_alt_text": "Preview image for \"{site}\"",

	"npf_renderer_link_block_fallback_embeds_are_disabled": "Embeds are disabled",

	"npf_renderer_error_video_link_block_fallback_heading": "Error: unable to render video block",
	"npf_renderer_video_link_block_fallback_description":   "Please click me to watch on the original site",
	"npf_renderer_error_link_block_fallback_native_video_player_non_tumblr_source": "Error: non-tumblr " +
		"source for video player",
	"npf_renderer_fallback_audio_block_thumbnail_alt_text": "Album art",
	"npf_renderer_error_audio_link_block_fallback_heading": "Error: unable to render audio block",
	"npf_renderer_audio_link_block_fallback_description":   "Please click me to listen on the original site",
	"npf_renderer_error_link_block_fallback_native_audio_player_non_tumblr_source": "Error: non-tumblr " +
		"source for audio player",

	"npf_renderer_poll_remaining_time": "{duration} remaining",
	"npf_renderer_poll_ended_on":       "Ended on: {ended_date}",
	"npf_renderer_post_attribution":    "From {author}",
	"npf_renderer_blog_attribution":    "Created by {author}",
	"npf_renderer_app_attribution":     "View on {platform}",
	"npf_renderer_unsupported_attribution": "Attributed via an unsupported (\"{attributee}\") " +
		"attribution type. Please report this over at " +
		"https://github.com/syeopite/npf-renderer",

	"alert_partial_restricted_results_heading": "Quick Update on Your Search Results",
	"alert_partial_restricted_results_message": "These search results have been partially restricted " +
		"due to Tumblr's content guidelines. You can expect " +
		"to see some empty results here sporadically.",
	"alert_restricted_results_heading": "Restricted search results",
	"alert_restricted_results_message": "These search results have been blocked by Tumblr for " +
		"violating their content guidelines.",
	"alert_view_post_on_account_restricted_blog_heading": "This blog requires an account to view",
	"alert_view_post_on_account_restricted_blog_message": "Only individual posts like this one can be " +
		"displayed",
	"alert_error_on_rendering_post_contents_heading": "Failed to render the contents of this post",
	"alert_error_on_rendering_post_contents": "The contents of this post has failed to render due to " +
		"an error. Check below for more information",
}

// Plurals maps message ids to (singular, plural) pairs.
var Plurals = map[string][2]string{
	"post_note_count":               {"{0} note", "{0} notes"},
	"npf_renderer_poll_total_votes": {"{votes} vote", "{votes} votes"},
}

// Translate returns the string for id, selecting the plural form when number
// is non-nil and the id has plural forms, then applying substitution.
func Translate(id string, number *int, substitution map[string]string) string {
	text, found := Strings[id]
	if number != nil {
		if pair, ok := Plurals[id]; ok {
			if *number == 1 {
				text = pair[0]
			} else {
				text = pair[1]
			}
			found = true
		}
	}
	if !found {
		text = id
	}
	for k, v := range substitution {
		text = strings.ReplaceAll(text, "{"+k+"}", v)
	}
	return text
}

// Conv is a convenience for template code: T(id, subst...).
func T(id string, substitution map[string]string) string {
	return Translate(id, nil, substitution)
}
