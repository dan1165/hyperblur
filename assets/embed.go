// Package assets embeds openblur's static front-end files so the binary is
// fully self-contained.
package assets

import "embed"

// FS holds the embedded stylesheets, scripts, fonts, images and robots.txt.
//
//go:embed css js images fonts robots.txt
var FS embed.FS
