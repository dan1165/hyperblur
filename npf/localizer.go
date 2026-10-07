package npf

import (
	"fmt"
	"strings"
	"time"
)

// Localizer supplies the human-readable strings and value formatting used
// while rendering. It mirrors the dict-based localizer of npf-renderer.
type Localizer interface {
	// Translate returns the (optionally substituted) string for key.
	Translate(key string, subst map[string]string) string
	// TranslatePlural returns the correct plural form for number.
	TranslatePlural(key string, number int, subst map[string]string) string
	FormatDuration(key string, d time.Duration) string
	FormatDatetime(key string, t time.Time) string
	FormatDecimal(key string, n float64) string
}

// DefaultLocalizer reproduces npf-renderer's built-in English strings.
type DefaultLocalizer struct{}

var defaultStrings = map[string]string{
	"asker_with_no_attribution": "Anonymous",
	"asker_and_ask_verb":        "{name} asked:",

	"unsupported_block_header": "Unsupported NPF block",
	"unsupported_block_description": "Placeholder for the unsupported \"{block}\" type NPF block" +
		"    Please report me over at https://github.com/syeopite/npf-renderer",

	"generic_image_alt_text":     "image",
	"link_block_poster_alt_text": "Preview image for \"{site}\"",

	"link_block_fallback_embeds_are_disabled": "Embeds are disabled",

	"error_video_link_block_fallback_heading":                         "Error: unable to render video block",
	"video_link_block_fallback_description":                           "Please click me to watch on the original site",
	"error_link_block_fallback_native_video_player_non_tumblr_source": "Error: non-tumblr source for video player",
	"fallback_audio_block_thumbnail_alt_text":                         "Album art",
	"error_audio_link_block_fallback_heading":                         "Error: unable to render audio block",
	"audio_link_block_fallback_description":                           "Please click me to listen on the original site",
	"error_link_block_fallback_native_audio_player_non_tumblr_source": "Error: non-tumblr source for audio player",

	"poll_remaining_time": "Remaining time: {duration}",
	"poll_ended_on":       "Ended on: {ended_date}",

	"post_attribution": "From {author}",
	"blog_attribution": "Created by {author}",
	"app_attribution":  "View on {platform}",

	"unsupported_attribution": "Attributed via an unsupported (\"{attributee}\") attribution type. " +
		"Please report this over at https://github.com/syeopite/npf-renderer",
}

func substitute(s string, subst map[string]string) string {
	for k, v := range subst {
		s = strings.ReplaceAll(s, "{"+k+"}", v)
	}
	return s
}

// Translate implements Localizer.
func (DefaultLocalizer) Translate(key string, subst map[string]string) string {
	return substitute(defaultStrings[key], subst)
}

// TranslatePlural implements Localizer. The default localizer has no real
// plural rules; npf-renderer's built-in is a constant "{votes} votes".
func (DefaultLocalizer) TranslatePlural(key string, number int, subst map[string]string) string {
	switch key {
	case "plural_poll_total_votes", "poll_total_votes":
		return substitute("{votes} votes", subst)
	}
	return substitute(defaultStrings[key], subst)
}

// FormatDuration implements Localizer.
func (DefaultLocalizer) FormatDuration(key string, d time.Duration) string {
	return formatTimedelta(d)
}

// FormatDatetime implements Localizer.
func (DefaultLocalizer) FormatDatetime(key string, t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// FormatDecimal implements Localizer.
func (DefaultLocalizer) FormatDecimal(key string, n float64) string {
	if n == float64(int64(n)) {
		return fmt.Sprintf("%d", int64(n))
	}
	return pyFloatStr(n)
}

// formatTimedelta renders a duration the way Python's str(timedelta) does,
// e.g. "7 days, 0:00:00" or "1 day, 2:03:04".
func formatTimedelta(d time.Duration) string {
	negative := d < 0
	if negative {
		d = -d
	}
	total := int64(d / time.Second)
	days := total / 86400
	rem := total % 86400
	hours := rem / 3600
	minutes := (rem % 3600) / 60
	seconds := rem % 60

	var b strings.Builder
	if days != 0 {
		if days == 1 {
			b.WriteString("1 day")
		} else {
			fmt.Fprintf(&b, "%d days", days)
		}
		if hours != 0 || minutes != 0 || seconds != 0 {
			b.WriteString(", ")
		}
	}
	if days != 0 || hours != 0 || minutes != 0 || seconds != 0 {
		fmt.Fprintf(&b, "%d:%02d:%02d", hours, minutes, seconds)
	}
	if negative {
		b.WriteString(" ago")
	}
	return b.String()
}

// pyFloatStr renders a float the way Python's str() would, so a whole number
// keeps a trailing ".0".
func pyFloatStr(f float64) string {
	s := fmt.Sprintf("%v", f)
	if !strings.ContainsAny(s, ".eE") {
		s += ".0"
	}
	return s
}
