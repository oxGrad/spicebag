// internal/dashboard/handlers_cv.go
package dashboard

import (
	"html/template"
	"net/http"
	"regexp"
	"strings"

	"github.com/oxGrad/spicebag/internal/fs"
)

type cvListData struct {
	Files []fs.FileInfo
}

type cvViewData struct {
	Name          string
	Content       template.HTML
	Themes        []string
	SelectedTheme string
	ThemeStyle    template.HTML
}

var cssRuleRe = regexp.MustCompile(`([^{]+)\{([^}]*)\}`)

// scopeCSS prefixes every selector in css with scope so theme styles only
// affect the preview container. "body" is mapped to scope itself.
func scopeCSS(css, scope string) string {
	return cssRuleRe.ReplaceAllStringFunc(css, func(match string) string {
		idx := strings.Index(match, "{")
		if idx < 0 {
			return match
		}
		rawSel := strings.TrimSpace(match[:idx])
		decl := match[idx:]
		parts := strings.Split(rawSel, ",")
		for i, s := range parts {
			s = strings.TrimSpace(s)
			if s == "body" {
				parts[i] = scope
			} else {
				parts[i] = scope + " " + s
			}
		}
		return strings.Join(parts, ", ") + " " + decl
	})
}

func (s *Server) handleCVList(w http.ResponseWriter, r *http.Request) {
	files, err := fs.ListCVs(s.root)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.render(w, "cv_list.html", cvListData{Files: files})
}

func (s *Server) handleCVView(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	content, err := fs.ReadCV(s.root, name)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	themes, _ := fs.ListThemes(s.root)
	selectedTheme := r.URL.Query().Get("theme")

	var themeStyle template.HTML
	if selectedTheme != "" {
		if css, err := fs.ReadTheme(s.root, selectedTheme); err == nil {
			scoped := scopeCSS(css, "#preview")
			themeStyle = template.HTML("<style>" + strings.ReplaceAll(scoped, "</style>", `<\/style>`) + "</style>")
		}
	}

	s.render(w, "cv_view.html", cvViewData{
		Name:          name,
		Content:       RenderMarkdown(content),
		Themes:        themes,
		SelectedTheme: selectedTheme,
		ThemeStyle:    themeStyle,
	})
}
