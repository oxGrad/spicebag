// Package assets embeds static assets into the binary.
package assets

import "embed"

//go:embed themes
var ThemesFS embed.FS
