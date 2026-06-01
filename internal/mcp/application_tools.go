// internal/mcp/application_tools.go
package mcp

import (
	"context"
	"fmt"

	"github.com/graditya/prospector/internal/db"
	"github.com/graditya/prospector/internal/fs"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerApplicationTools() {
	s.mcpSrv.AddTool(
		mcplib.NewTool("create_application",
			mcplib.WithDescription("Create full application folder with CV, cover letter, job post, and metadata"),
			mcplib.WithString("company", mcplib.Required(), mcplib.Description("Company name")),
			mcplib.WithString("role", mcplib.Required(), mcplib.Description("Role/job title")),
			mcplib.WithString("date", mcplib.Required(), mcplib.Description("Application date in YYYY-MM-DD format")),
			mcplib.WithString("cv_content", mcplib.Required(), mcplib.Description("Markdown content for the CV")),
			mcplib.WithString("cover_letter_content", mcplib.Required(), mcplib.Description("Markdown content for the cover letter")),
			mcplib.WithString("job_post_content", mcplib.Required(), mcplib.Description("Markdown content for the job post")),
			mcplib.WithString("base_cv_used", mcplib.Description("Base CV filename used as source")),
			mcplib.WithString("notes", mcplib.Description("Optional notes about the application")),
		),
		func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
			company := req.GetString("company", "")
			role := req.GetString("role", "")
			date := req.GetString("date", "")
			cvContent := req.GetString("cv_content", "")
			coverLetterContent := req.GetString("cover_letter_content", "")
			jobPostContent := req.GetString("job_post_content", "")
			baseCVUsed := req.GetString("base_cv_used", "")
			notes := req.GetString("notes", "")

			folderPath, err := fs.CreateApplication(s.root, fs.ApplicationRequest{
				Company:            company,
				Role:               role,
				Date:               date,
				CVContent:          cvContent,
				CoverLetterContent: coverLetterContent,
				JobPostContent:     jobPostContent,
				BaseCVUsed:         baseCVUsed,
				Notes:              notes,
			})
			if err != nil {
				return mcplib.NewToolResultError(fmt.Sprintf("creating application: %v", err)), nil
			}

			appID, err := s.store.UpsertApplication(db.Application{
				Company:     company,
				Role:        role,
				AppliedDate: date,
				BaseCVUsed:  baseCVUsed,
				Notes:       notes,
				FolderPath:  folderPath,
			})
			if err != nil {
				return mcplib.NewToolResultError(fmt.Sprintf("saving application to DB: %v", err)), nil
			}

			if err := s.store.AddStatusHistory(appID, "applied", ""); err != nil {
				return mcplib.NewToolResultError(fmt.Sprintf("recording status history: %v", err)), nil
			}

			return mcplib.NewToolResultText(fmt.Sprintf("Created application at %s", folderPath)), nil
		},
	)
}
