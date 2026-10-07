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

// pyFloatStr renders a float the way Python's str() would, so a whole number
// keeps a trailing ".0".
func pyFloatStr(f float64) string {
	s := fmt.Sprintf("%v", f)
	if !strings.ContainsAny(s, ".eE") {
		s += ".0"
	}
	return s
}
