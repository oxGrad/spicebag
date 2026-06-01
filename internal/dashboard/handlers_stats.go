// internal/dashboard/handlers_stats.go
package dashboard

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/graditya/prospector/internal/db"
	"github.com/graditya/prospector/internal/fs"
	"github.com/graditya/prospector/internal/parser"
)

type statsPageData struct {
	Stats db.ExperienceStats
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.store.GetExperienceStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// stats.html embeds {{template "stats_content" .Stats}} so parse both files.
	t, err := template.ParseFS(templateFS, "templates/layout.html", "templates/stats.html", "templates/stats_content.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "layout", statsPageData{Stats: stats}); err != nil {
		log.Printf("handleStats: %v", err)
	}
}

func (s *Server) handleStatsSync(w http.ResponseWriter, r *http.Request) {
	cvFiles, err := fs.ListCVs(s.root)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, f := range cvFiles {
		content, err := fs.ReadCV(s.root, f.Name)
		if err != nil {
			continue
		}
		if err := s.store.DeleteExperienceBySyncedFrom(f.Name); err != nil {
			http.Error(w, fmt.Sprintf("delete old entries: %v", err), http.StatusInternalServerError)
			return
		}
		entries, err := parser.ParseExperience(content)
		if err != nil {
			continue
		}
		dbEntries := make([]db.ExperienceEntry, len(entries))
		for i, e := range entries {
			dbEntries[i] = db.ExperienceEntry{
				RoleType: e.RoleType, Company: e.Company,
				StartDate: e.StartDate, EndDate: e.EndDate, SyncedFrom: f.Name,
			}
		}
		if len(dbEntries) > 0 {
			s.store.UpsertExperience(dbEntries)
		}
	}

	stats, err := s.store.GetExperienceStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.renderPartial(w, "stats_content.html", "stats_content", stats)
}
