// internal/mcp/application_tools.go
package mcp

import (
	"context"
	"fmt"

	"github.com/oxGrad/spicebag/internal/db"
	"github.com/oxGrad/spicebag/internal/fs"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerApplicationTools() {
	s.mcpSrv.AddTool(
		mcplib.NewTool(
			"create_application",
			mcplib.WithDescription("Create full application folder with CV, cover letter, job post, and metadata"),
			mcplib.WithString("company", mcplib.Required(), mcplib.Description("Company name")),
			mcplib.WithString("role", mcplib.Required(), mcplib.Description("Role/job title")),
			mcplib.WithString("date", mcplib.Required(), mcplib.Description("Application date in YYYY-MM-DD format")),
			mcplib.WithString("cv_content", mcplib.Required(), mcplib.Description("Markdown content for the CV")),
			mcplib.WithString("cover_letter_content", mcplib.Required(), mcplib.Description("Markdown content for the cover letter")),
			mcplib.WithString("job_post_content", mcplib.Required(), mcplib.Description("Markdown content for the job post")),
			mcplib.WithString("base_cv_used", mcplib.Description("Base CV filename used as source")),
			mcplib.WithString("notes", mcplib.Description("Optional notes about the application")),
			mcplib.WithString("job_url", mcplib.Description("Original URL of the job post, if sourced from a URL")),
			mcplib.WithString("job_summary", mcplib.Description("2-3 sentence summary of the role and key requirements")),
			mcplib.WithNumber("match_score", mcplib.Description("CV-to-job match percentage 0-100 from tailoring assessment")),
			mcplib.WithString("tailoring_notes", mcplib.Description("Tailoring assessment: strengths, gaps, and plan from step 3")),
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
			jobURL := req.GetString("job_url", "")
			jobSummary := req.GetString("job_summary", "")
			tailoringNotes := req.GetString("tailoring_notes", "")
			var matchScore *int
			if _, ok := req.GetArguments()["match_score"]; ok {
				v := req.GetInt("match_score", 0)
				matchScore = &v
			}

			folderPath, err := fs.CreateApplication(s.root, fs.ApplicationRequest{
				Company:            company,
				Role:               role,
				Date:               date,
				CVContent:          cvContent,
				CoverLetterContent: coverLetterContent,
				JobPostContent:     jobPostContent,
				BaseCVUsed:         baseCVUsed,
				Notes:              notes,
				JobURL:             jobURL,
				JobSummary:         jobSummary,
			})
			if err != nil {
				return mcplib.NewToolResultError(fmt.Sprintf("creating application: %v", err)), nil
			}

			id, err := s.store.UpsertApplication(db.Application{
				Company:        company,
				Role:           role,
				AppliedDate:    "", // set manually by user when they actually submit
				BaseCVUsed:     baseCVUsed,
				Notes:          notes,
				FolderPath:     folderPath,
				JobURL:         jobURL,
				JobSummary:     jobSummary,
				MatchScore:     matchScore,
				TailoringNotes: tailoringNotes,
			})
			if err != nil {
				return mcplib.NewToolResultError(fmt.Sprintf("saving application to DB: %v", err)), nil
			}

			if err := s.store.AddStatusHistory(id, "pending", ""); err != nil {
				return mcplib.NewToolResultError(fmt.Sprintf("setting initial status: %v", err)), nil
			}

			if jobURL != "" {
				s.store.LinkApplicationToScrapedJob(id, jobURL) //nolint:errcheck
			}

			return mcplib.NewToolResultText(fmt.Sprintf("Created application at %s", folderPath)), nil
		},
	)
}
