// internal/dashboard/handlers_themes.go
package dashboard

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/oxGrad/spicebag/internal/fs"
	"github.com/oxGrad/spicebag/internal/pdf"
)

func (s *Server) handleAPIThemesList(w http.ResponseWriter, r *http.Request) {
	themes, _ := fs.ListThemes(s.root)
	if themes == nil {
		themes = []string{}
	}
	writeJSON(w, themes)
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
	if err := os.MkdirAll(themeDir, 0o755); err != nil {
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
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, map[string]string{"name": strings.TrimSuffix(name, ".css")})
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

	// read the HTML fragment directly
	fragment, err := os.ReadFile(resolved)
	if err != nil {
		http.Error(w, fmt.Sprintf("read file: %v", err), http.StatusNotFound)
		return
	}

	css := s.themeCSS(theme)
	client := pdf.NewClient(s.cfg.GotenbergURL)
	pdfBytes, err := client.RenderPDF(string(fragment), css)
	if err != nil {
		http.Error(w, fmt.Sprintf("render PDF: %v", err), http.StatusInternalServerError)
		return
	}

	// derive filename for download; sanitize for Content-Disposition header
	base := filepath.Base(filePath)
	pdfName := strings.TrimSuffix(base, filepath.Ext(base)) + ".pdf"
	pdfName = strings.Map(func(r rune) rune {
		if r == '"' || r == '\n' || r == '\r' {
			return '_'
		}
		return r
	}, pdfName)

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, pdfName))
	w.Write(pdfBytes)
}
