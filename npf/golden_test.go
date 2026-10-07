package npf

import (
	"encoding/json"
	"os"
	"sort"
	"testing"
)

type goldenCase struct {
	Content []any `json:"content"`
	Layout  []any `json:"layout"`
	Opts    struct {
		URLPrefix     string `json:"url_prefix"`
		ForbidIframes bool   `json:"forbid_iframes"`
		Truncate      bool   `json:"truncate"`
	} `json:"opts"`
	Error bool   `json:"error"`
	HTML  string `json:"html"`
}

// TestGolden checks the Go port against output produced by the reference
// npf_renderer. Regenerate testdata/golden.json with testdata/gen_golden.py.
func TestGolden(t *testing.T) {
	data, err := os.ReadFile("testdata/golden.json")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	var cases map[string]goldenCase
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
			opts := Options{
				Truncate:              tc.Opts.Truncate,
				ForbidExternalIframes: tc.Opts.ForbidIframes,
			}
			if tc.Opts.URLPrefix != "" {
				prefix := tc.Opts.URLPrefix
				opts.URLHandler = func(u string) string { return prefix + u }
			}

			hasError, html := Format(tc.Content, tc.Layout, opts)
			if hasError != tc.Error {
				t.Errorf("error flag: got %v, want %v", hasError, tc.Error)
			}
			if html != tc.HTML {
				t.Errorf("html mismatch\n got: %s\nwant: %s", html, tc.HTML)
			}
		})
	}
}
