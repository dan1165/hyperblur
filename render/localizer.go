package render

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// Localizer provides hyperblur's English strings and formatting to the
// npf.Localizer interface.
type Localizer struct{}

// Translate implements npf.Localizer.
func (Localizer) Translate(key string, subst map[string]string) string {
	return substitute(npfString(key), subst)
}

// TranslatePlural implements npf.Localizer.
func (Localizer) TranslatePlural(key string, number int, subst map[string]string) string {
	if strings.TrimPrefix(key, "plural_") == "poll_total_votes" {
		text := "{votes} votes"
		if number == 1 {
			text = "{votes} vote"
		}
		return substitute(text, subst)
	}
	return substitute(npfString(strings.TrimPrefix(key, "plural_")), subst)
}

func substitute(text string, subst map[string]string) string {
	for k, v := range subst {
		text = strings.ReplaceAll(text, "{"+k+"}", v)
	}
	return text
}

// npfString holds the strings the NPF formatter asks the localizer for.
func npfString(key string) string {
	switch key {
	case "asker_with_no_attribution":
		return "Anonymous"
	case "asker_and_ask_verb":
		return "{name} asked"
	case "unsupported_block_header":
		return "Unsupported NPF block"
	case "unsupported_block_description":
		return "Placeholder for the unsupported \"{block}\" type NPF block Please report me over at https://github.com/syeopite/npf-renderer"
	case "generic_image_alt_text":
		return "image"
	case "link_block_poster_alt_text":
		return "Preview image for \"{site}\""
	case "link_block_fallback_embeds_are_disabled":
		return "Embeds are disabled"
	case "error_video_link_block_fallback_heading":
		return "Error: unable to render video block"
	case "video_link_block_fallback_description":
		return "Please click me to watch on the original site"
	case "error_link_block_fallback_native_video_player_non_tumblr_source":
		return "Error: non-tumblr source for video player"
	case "fallback_audio_block_thumbnail_alt_text":
		return "Album art"
	case "error_audio_link_block_fallback_heading":
		return "Error: unable to render audio block"
	case "audio_link_block_fallback_description":
		return "Please click me to listen on the original site"
	case "error_link_block_fallback_native_audio_player_non_tumblr_source":
		return "Error: non-tumblr source for audio player"
	case "poll_remaining_time":
		return "{duration} remaining"
	case "poll_ended_on":
		return "Ended on: {ended_date}"
	case "post_attribution":
		return "From {author}"
	case "blog_attribution":
		return "Created by {author}"
	case "app_attribution":
		return "View on {platform}"
	case "unsupported_attribution":
		return "Attributed via an unsupported (\"{attributee}\") attribution type. Please report this over at https://github.com/syeopite/npf-renderer"
	}
	return key
}

// FormatDuration implements npf.Localizer.
//
// ponytail: approximates babel's format_timedelta(threshold=1.1) by picking the
// largest unit whose value is >= 1.1. Swap for golang.org/x/text if the exact
// babel wording is ever needed.
func (Localizer) FormatDuration(key string, d time.Duration) string {
	units := []struct {
		seconds float64
		name    string
	}{
		{365 * 86400, "year"},
		{30 * 86400, "month"},
		{7 * 86400, "week"},
		{86400, "day"},
		{3600, "hour"},
		{60, "minute"},
		{1, "second"},
	}
	seconds := d.Seconds()
	for _, u := range units {
		value := seconds / u.seconds
		if value >= 1.1 {
			n := int(math.Round(value))
			if n == 1 {
				return "1 " + u.name
			}
			return fmt.Sprintf("%d %ss", n, u.name)
		}
	}
	return "0 seconds"
}

// FormatDatetime implements npf.Localizer. babel's en_US "short" format is
// M/d/yy, h:mm a with a narrow no-break space (U+202F) before AM/PM.
func (Localizer) FormatDatetime(key string, t time.Time) string {
	return t.Format("1/2/06, 3:04") + "\u202f" + t.Format("PM")
}

// FormatDecimal implements npf.Localizer with en_US thousands grouping.
func (Localizer) FormatDecimal(key string, n float64) string {
	negative := n < 0
	if negative {
		n = -n
	}

	s := strconv.FormatFloat(n, 'f', -1, 64)
	intPart := s
	frac := ""
	if i := strings.IndexByte(s, '.'); i >= 0 {
		intPart, frac = s[:i], s[i:]
	}

	var b strings.Builder
	for i, ch := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(ch)
	}
	if negative {
		return "-" + b.String() + frac
	}
	return b.String() + frac
}
