// internal/dashboard/handlers_cl.go
package dashboard

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/oxGrad/spicebag/internal/fs"
)

type clListData struct {
	Files []fs.FileInfo
}

type clViewData struct {
	Name          string
	Content       template.HTML
	Themes        []string
	SelectedTheme string
	ThemeStyle    template.HTML
}

func (s *Server) handleCLList(w http.ResponseWriter, r *http.Request) {
	files, err := fs.ListCoverLetters(s.root)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.render(w, "cl_list.html", clListData{Files: files})
}

func (s *Server) handleCLView(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	content, err := fs.ReadCoverLetter(s.root, name)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	themes, _ := fs.ListThemes(s.root)
	selectedTheme := r.URL.Query().Get("theme")

	var themeStyle template.HTML
	if selectedTheme != "" {
		if css, err := fs.ReadTheme(s.root, selectedTheme); err == nil {
			themeStyle = template.HTML("<style>" + strings.ReplaceAll(css, "</style>", `<\/style>`) + "</style>")
		}
	}

	s.render(w, "cl_view.html", clViewData{
		Name:          name,
		Content:       RenderMarkdown(content),
		Themes:        themes,
		SelectedTheme: selectedTheme,
		ThemeStyle:    themeStyle,
	})
}
