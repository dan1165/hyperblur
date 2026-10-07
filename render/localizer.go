package render

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/dan1165/openblur/i18n"
)

// Localizer bridges openblur's English strings and formatting to the
// npf.Localizer interface, mirroring i18n.NPFRendererLocalizer.
type Localizer struct{}

// Translate implements npf.Localizer.
func (Localizer) Translate(key string, subst map[string]string) string {
	return i18n.Translate("npf_renderer_"+key, nil, subst)
}

// TranslatePlural implements npf.Localizer.
func (Localizer) TranslatePlural(key string, number int, subst map[string]string) string {
	return i18n.Translate("npf_renderer_"+strings.TrimPrefix(key, "plural_"), &number, subst)
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
