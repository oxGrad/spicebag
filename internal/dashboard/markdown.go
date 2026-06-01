// internal/dashboard/markdown.go
package dashboard

import (
	"bytes"
	"html/template"

	"github.com/yuin/goldmark"
)

// RenderMarkdown converts markdown to safe HTML for embedding in templates.
func RenderMarkdown(md string) template.HTML {
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(md), &buf); err != nil {
		return template.HTML("<pre>" + template.HTMLEscapeString(md) + "</pre>")
	}
	return template.HTML(buf.String())
}
