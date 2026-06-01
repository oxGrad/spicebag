// Package assets embeds static assets (themes, skills) into the binary.
package assets

import "embed"

//go:embed themes
var ThemesFS embed.FS

//go:embed skills
var SkillsFS embed.FS
