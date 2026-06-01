// internal/mcp/pdf_tools.go
package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	pdfpkg "github.com/graditya/prospector/internal/pdf"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerPDFTools() {
	s.mcpSrv.AddTool(
		mcplib.NewTool("export_pdf",
			mcplib.WithDescription("Render a CV or cover letter to PDF via Gotenberg; returns the output file path"),
			mcplib.WithString("file_path", mcplib.Required(), mcplib.Description("Relative path to the markdown file to render")),
			mcplib.WithString("theme", mcplib.Required(), mcplib.Description("Theme name (CSS filename without extension)")),
		),
		func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
			filePath := req.GetString("file_path", "")
			theme := req.GetString("theme", "")

			mdBytes, err := os.ReadFile(filepath.Join(s.root, filePath))
			if err != nil {
				return mcplib.NewToolResultError(fmt.Sprintf("reading markdown: %v", err)), nil
			}

			cssBytes, err := os.ReadFile(filepath.Join(s.root, "themes", theme+".css"))
			if err != nil {
				return mcplib.NewToolResultError(fmt.Sprintf("reading theme CSS: %v", err)), nil
			}

			pdfBytes, err := pdfpkg.NewClient(s.gotURL).RenderPDF(string(mdBytes), string(cssBytes))
			if err != nil {
				return mcplib.NewToolResultError(fmt.Sprintf("rendering PDF: %v", err)), nil
			}

			outPath := filepath.Join(s.root, filePath[:len(filePath)-3]+".pdf")
			if err := os.WriteFile(outPath, pdfBytes, 0644); err != nil {
				return mcplib.NewToolResultError(fmt.Sprintf("writing PDF: %v", err)), nil
			}

			return mcplib.NewToolResultText(outPath), nil
		},
	)
}
