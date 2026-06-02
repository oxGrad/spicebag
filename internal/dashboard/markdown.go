package dashboard

import (
	"bytes"
	"html/template"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"golang.org/x/net/html"
)

var gm = goldmark.New(
	goldmark.WithExtensions(
		extension.Table,
		extension.Strikethrough,
	),
)

// RenderMarkdown converts markdown to safe HTML with CV-semantic wrapper divs.
func RenderMarkdown(src string) template.HTML {
	var buf bytes.Buffer
	if err := gm.Convert([]byte(src), &buf); err != nil {
		return template.HTML("<pre>" + template.HTMLEscapeString(src) + "</pre>")
	}
	return template.HTML(postProcess(buf.String()))
}

// postProcess wraps goldmark HTML in semantic divs so CSS can target CV structure
// without fragile positional selectors:
//
//	.cv-header  — name + contact paragraphs before the first <h1>
//	.cv-section — each <h1> and its content
//	.cv-entry   — each <table> (company/date row) plus everything until the next table or h1
func postProcess(htmlStr string) string {
	doc, err := html.Parse(strings.NewReader("<html><body>" + htmlStr + "</body></html>"))
	if err != nil {
		return htmlStr
	}

	body := findElement(doc, "body")
	if body == nil {
		return htmlStr
	}

	nodes := blockChildren(body)
	if len(nodes) == 0 {
		return htmlStr
	}

	var out strings.Builder
	i := 0

	// Header block: all nodes before the first <h1>
	var header []*html.Node
	for i < len(nodes) && !isTag(nodes[i], "h1") {
		header = append(header, nodes[i])
		i++
	}
	if len(header) > 0 {
		out.WriteString(`<div class="cv-header">`)
		for _, n := range header {
			renderNode(&out, n)
		}
		out.WriteString("</div>\n")
	}

	// Section blocks: one per <h1>
	for i < len(nodes) {
		if !isTag(nodes[i], "h1") {
			renderNode(&out, nodes[i])
			i++
			continue
		}

		out.WriteString(`<div class="cv-section">`)
		renderNode(&out, nodes[i])
		i++

		for i < len(nodes) && !isTag(nodes[i], "h1") {
			if isTag(nodes[i], "table") {
				// Entry block: table + role line + bullets, until next table or h1
				out.WriteString(`<div class="cv-entry">`)
				for i < len(nodes) && !isTag(nodes[i], "h1") {
					renderNode(&out, nodes[i])
					i++
					// Close entry when we hit the next table
					if i < len(nodes) && isTag(nodes[i], "table") {
						break
					}
				}
				out.WriteString("</div>\n")
			} else {
				renderNode(&out, nodes[i])
				i++
			}
		}

		out.WriteString("</div>\n")
	}

	return out.String()
}

func blockChildren(parent *html.Node) []*html.Node {
	var nodes []*html.Node
	for c := parent.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode && strings.TrimSpace(c.Data) == "" {
			continue
		}
		nodes = append(nodes, c)
	}
	return nodes
}

func findElement(n *html.Node, tag string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findElement(c, tag); found != nil {
			return found
		}
	}
	return nil
}

func isTag(n *html.Node, tag string) bool {
	return n.Type == html.ElementNode && n.Data == tag
}

func renderNode(w *strings.Builder, n *html.Node) {
	var buf bytes.Buffer
	_ = html.Render(&buf, n)
	w.Write(buf.Bytes())
}
