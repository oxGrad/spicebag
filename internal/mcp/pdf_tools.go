// internal/mcp/pdf_tools.go
package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	pdfpkg "github.com/oxGrad/spicebag/internal/pdf"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerPDFTools() {
	s.mcpSrv.AddTool(
		mcplib.NewTool(
			"export_pdf",
			mcplib.WithDescription("Render a CV or cover letter HTML file to PDF; returns the output file path"),
			mcplib.WithString("file_path", mcplib.Required(), mcplib.Description("Relative path to the HTML file to render (e.g. cv/base.html)")),
			mcplib.WithString("theme", mcplib.Required(), mcplib.Description("Theme name (CSS filename without extension)")),
		),
		func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
			filePath := req.GetString("file_path", "")
			theme := req.GetString("theme", "")

			ext := filepath.Ext(filePath)
			if ext != ".html" && ext != ".md" {
				return mcplib.NewToolResultError(fmt.Sprintf("file_path must be an .html file, got %q", filePath)), nil
			}

			htmlBytes, err := os.ReadFile(filepath.Join(s.root, filePath))
			if err != nil {
				return mcplib.NewToolResultError(fmt.Sprintf("reading file: %v", err)), nil
			}

			var css string
			if theme != "" {
				cssBytes, err := os.ReadFile(filepath.Join(s.root, "themes", theme+".css"))
				if err != nil {
					return mcplib.NewToolResultError(fmt.Sprintf("reading theme CSS: %v", err)), nil
				}
				css = string(cssBytes)
			}

			pdfBytes, err := pdfpkg.RenderWithFallback(s.gotURL, string(htmlBytes), css)
			if err != nil {
				return mcplib.NewToolResultError(fmt.Sprintf("rendering PDF: %v", err)), nil
			}

			outPath := filepath.Join(s.root, strings.TrimSuffix(filePath, ext)+".pdf")
			if err := os.WriteFile(outPath, pdfBytes, 0o644); err != nil {
				return mcplib.NewToolResultError(fmt.Sprintf("writing PDF: %v", err)), nil
			}

			return mcplib.NewToolResultText(outPath), nil
		},
	)
}
