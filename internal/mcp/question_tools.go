package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	mcplib "github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerQuestionTools() {
	s.mcpSrv.AddTool(
		mcplib.NewTool(
			"list_application_questions",
			mcplib.WithDescription("List questions (and any existing answers) for an application"),
			mcplib.WithNumber("application_id", mcplib.Required(), mcplib.Description("Application DB integer ID")),
		),
		func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
			id := int64(req.GetFloat("application_id", 0))
			qs, err := s.store.ListQuestions(id)
			if err != nil {
				return mcplib.NewToolResultError(err.Error()), nil
			}
			if qs == nil {
				return mcplib.NewToolResultText("[]"), nil
			}
			out, _ := json.Marshal(qs)
			return mcplib.NewToolResultText(string(out)), nil
		},
	)

	s.mcpSrv.AddTool(
		mcplib.NewTool(
			"write_application_answers",
			mcplib.WithDescription("Save crafted answers for application form questions"),
			mcplib.WithNumber("application_id", mcplib.Required(), mcplib.Description("Application DB integer ID")),
			mcplib.WithString("answers", mcplib.Required(), mcplib.Description(`JSON array of {"question_id": <id>, "answer": "<text>"}`)),
		),
		func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
			id := int64(req.GetFloat("application_id", 0))
			answersJSON := req.GetString("answers", "")

			var entries []struct {
				QuestionID int64  `json:"question_id"`
				Answer     string `json:"answer"`
			}
			if err := json.Unmarshal([]byte(answersJSON), &entries); err != nil {
				return mcplib.NewToolResultError(fmt.Sprintf("invalid answers JSON: %v", err)), nil
			}

			pairs := make([]struct{ QuestionID int64; Answer string }, len(entries))
			for i, e := range entries {
				pairs[i] = struct{ QuestionID int64; Answer string }{e.QuestionID, e.Answer}
			}
			if err := s.store.BulkUpdateAnswers(pairs); err != nil {
				return mcplib.NewToolResultError(err.Error()), nil
			}
			return mcplib.NewToolResultText(fmt.Sprintf("Saved %d answers for application %d", len(entries), id)), nil
		},
	)
}
