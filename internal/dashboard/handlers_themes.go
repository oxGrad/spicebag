// internal/dashboard/handlers_themes.go
package dashboard

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/graditya/prospector/internal/fs"
	"github.com/graditya/prospector/internal/pdf"
)

type themesPageData struct {
	Themes         []string
	PreviewTheme   string
	PreviewStyle   template.HTML
	PreviewContent template.HTML
}

func (s *Server) handleThemesList(w http.ResponseWriter, r *http.Request) {
	themes, _ := fs.ListThemes(s.root)
	s.render(w, "themes.html", themesPageData{Themes: themes})
}

func (s *Server) handleThemePreview(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	css, err := fs.ReadTheme(s.root, name)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// render preview with the first available CV, or sample text
	previewMD := "# Sample Heading\n\nThis is how your theme looks with **bold**, *italic*, and regular text.\n\n## Section\n\nMore content here."
	cvParam := r.URL.Query().Get("cv")
	if cvParam != "" {
		if content, err := fs.ReadCV(s.root, cvParam); err == nil {
			previewMD = content
		}
	}

	themes, _ := fs.ListThemes(s.root)
	s.render(w, "themes.html", themesPageData{
		Themes:         themes,
		PreviewTheme:   name,
		PreviewStyle:   template.HTML("<style>" + css + "</style>"),
		PreviewContent: RenderMarkdown(previewMD),
	})
}

func (s *Server) handleThemeUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		http.Error(w, "parse form: "+err.Error(), http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("theme")
	if err != nil {
		http.Error(w, "missing file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	name := filepath.Base(header.Filename) // strip any directory components
	if !strings.HasSuffix(name, ".css") {
		http.Error(w, "file must be a .css file", http.StatusBadRequest)
		return
	}

	themeDir := filepath.Join(s.root, "themes")
	if err := os.MkdirAll(themeDir, 0755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	dest := filepath.Join(themeDir, name)
	f, err := os.Create(dest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	if _, err := io.Copy(f, file); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/themes", http.StatusSeeOther)
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "parse form: "+err.Error(), http.StatusBadRequest)
		return
	}
	filePath := r.FormValue("file_path")
	theme := r.FormValue("theme")

	if filePath == "" {
		http.Error(w, "file_path is required", http.StatusBadRequest)
		return
	}

	// guard against path traversal
	rootClean := filepath.Clean(s.root)
	resolved := filepath.Join(rootClean, filePath)
	if !strings.HasPrefix(resolved, rootClean+string(os.PathSeparator)) {
		http.Error(w, "invalid file_path", http.StatusBadRequest)
		return
	}

	// read the markdown file
	mdBytes, err := os.ReadFile(resolved)
	if err != nil {
		http.Error(w, fmt.Sprintf("read file: %v", err), http.StatusNotFound)
		return
	}

	// render markdown to HTML
	htmlContent := string(RenderMarkdown(string(mdBytes)))

	// read theme CSS (optional)
	var css string
	if theme != "" {
		if cssBytes, err := fs.ReadTheme(s.root, theme); err == nil {
			css = cssBytes
		}
	}

	client := pdf.NewClient(s.cfg.GotenbergURL)
	pdfBytes, err := client.RenderPDF(htmlContent, css)
	if err != nil {
		http.Error(w, fmt.Sprintf("render PDF: %v", err), http.StatusInternalServerError)
		return
	}

	// derive filename for download
	base := filepath.Base(filePath)
	pdfName := strings.TrimSuffix(base, filepath.Ext(base)) + ".pdf"

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, pdfName))
	w.Write(pdfBytes)
}
