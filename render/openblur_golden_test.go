package render

import (
	"encoding/json"
	"os"
	"sort"
	"testing"
	"time"
)

type openblurCase struct {
	Content      []any  `json:"content"`
	Layout       []any  `json:"layout"`
	BlogName     string `json:"blog_name"`
	PostID       string `json:"post_id"`
	ErrorName    string `json:"error_name"`
	ErrorMessage string `json:"error_message"`
	HTML         string `json:"html"`
}

// TestOpenblurGolden checks the Go openblur render layer against output from
// openblur's own Python renderer. Regenerate with testdata/gen_openblur_golden.py.
func TestOpenblurGolden(t *testing.T) {
	data, err := os.ReadFile("testdata/openblur_golden.json")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	var cases map[string]openblurCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatalf("parse golden: %v", err)
	}

	names := make([]string, 0, len(cases))
	for name := range cases {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		tc := cases[name]
		t.Run(name, func(t *testing.T) {
			renderErr, html := FormatNPF(tc.Content, tc.Layout, Params{
				ExpandPosts: true,
				BlogName:    tc.BlogName,
				PostID:      tc.PostID,
			})

			if tc.ErrorName != "" {
				if renderErr == nil {
					t.Fatalf("expected render error %q, got none", tc.ErrorName)
				}
				if renderErr.Name != tc.ErrorName {
					t.Errorf("error name: got %q, want %q", renderErr.Name, tc.ErrorName)
				}
				if renderErr.Message != tc.ErrorMessage {
					t.Errorf("error message: got %q, want %q", renderErr.Message, tc.ErrorMessage)
				}
			} else if renderErr != nil {
				t.Fatalf("unexpected render error: %+v", renderErr)
			}

			if html != tc.HTML {
				t.Errorf("html mismatch\n got: %s\nwant: %s", html, tc.HTML)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		seconds float64
		want    string
	}{
		{30, "30 seconds"},
		{90, "2 minutes"},
		{3600, "60 minutes"},
		{5400, "2 hours"},
		{86400, "24 hours"},
		{7 * 86400, "7 days"},
		{8 * 86400, "1 week"},
		{13 * 86400, "2 weeks"},
		{40 * 86400, "1 month"},
	}
	for _, c := range cases {
		got := (Localizer{}).FormatDuration("poll_duration", time.Duration(c.seconds*float64(time.Second)))
		if got != c.want {
			t.Errorf("%v seconds: got %q, want %q", c.seconds, got, c.want)
		}
	}
}
