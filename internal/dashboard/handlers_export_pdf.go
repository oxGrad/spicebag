package dashboard

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/oxGrad/spicebag/internal/pdf"
)

type PDFFile struct {
	Filename string `json:"filename"`
	DocType  string `json:"doc_type"` // "cv", "cl", "base", or ""
}

func inferDocType(filename string) string {
	lower := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(lower, "-cover-letter.pdf"):
		return "cl"
	case strings.HasSuffix(lower, "-base-cv.pdf"):
		return "base"
	case strings.HasSuffix(lower, "-cv.pdf"):
		return "cv"
	default:
		return ""
	}
}

func (s *Server) handleAPIAppListPDFs(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	app, err := s.store.GetApplicationByID(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	pdfDir := filepath.Join(s.root, "applications", app.FolderPath, "pdf")
	entries, err := os.ReadDir(pdfDir)
	if os.IsNotExist(err) {
		writeJSON(w, []PDFFile{})
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var pdfs []PDFFile
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".pdf") {
			pdfs = append(pdfs, PDFFile{Filename: e.Name(), DocType: inferDocType(e.Name())})
		}
	}
	if pdfs == nil {
		pdfs = []PDFFile{}
	}
	writeJSON(w, pdfs)
}

func (s *Server) handleRenderAppPDF(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	app, err := s.store.GetApplicationByID(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	filename := r.PathValue("filename")
	pdfDir := filepath.Join(s.root, "applications", app.FolderPath, "pdf")
	resolved := filepath.Join(pdfDir, filename)
	if !strings.HasPrefix(resolved, filepath.Clean(pdfDir)+string(os.PathSeparator)) {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	data, err := os.ReadFile(resolved)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Write(data)
}

var (
	h1Re  = regexp.MustCompile(`(?i)<h1[^>]*>(.*?)</h1>`)
	tagRe = regexp.MustCompile(`<[^>]+>`)
)

func extractH1(html string) string {
	if m := h1Re.FindStringSubmatch(html); len(m) > 1 {
		return strings.TrimSpace(tagRe.ReplaceAllString(m[1], ""))
	}
	return ""
}

func pdfSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	// collapse multiple dashes
	return regexp.MustCompile(`-{2,}`).ReplaceAllString(b.String(), "-")
}

func (s *Server) handleAPIAppExportPDF(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	app, err := s.store.GetApplicationByID(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	docType := r.FormValue("doc_type") // "cv", "cl", "base"
	theme := r.FormValue("theme")

	appBase := filepath.Join(s.root, "applications")
	base := filepath.Clean(appBase)

	var srcPath string
	var pdfSuffix string
	switch docType {
	case "cv":
		srcPath = filepath.Join(base, app.FolderPath, "cv.html")
		pdfSuffix = "cv"
	case "cl":
		// try .html first, fall back to .md for older applications
		srcPath = filepath.Join(base, app.FolderPath, "cover-letter.html")
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			srcPath = filepath.Join(base, app.FolderPath, "cover-letter.md")
		}
		pdfSuffix = "cover-letter"
	case "base":
		if app.BaseCVUsed == "" {
			http.Error(w, "no base CV recorded for this application", http.StatusBadRequest)
			return
		}
		srcPath = filepath.Join(s.root, "cv", app.BaseCVUsed)
		pdfSuffix = "base-cv"
	default:
		http.Error(w, "doc_type must be cv, cl, or base", http.StatusBadRequest)
		return
	}

	fragment, err := os.ReadFile(srcPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("read document: %v", err), http.StatusNotFound)
		return
	}

	var css string
	if theme != "" {
		cssPath := filepath.Join(s.root, "themes", theme+".css")
		if b, err := os.ReadFile(cssPath); err == nil {
			css = string(b)
		}
	}

	pdfBytes, err := pdf.RenderWithFallback(s.cfg.GotenbergURL, string(fragment), css)
	if err != nil {
		http.Error(w, fmt.Sprintf("render PDF: %v", err), http.StatusInternalServerError)
		return
	}

	// derive person name from h1, fall back to "candidate"
	name := extractH1(string(fragment))
	if name == "" {
		name = "candidate"
	}

	pdfName := fmt.Sprintf("%s-%s-%s.pdf", pdfSlug(name), pdfSlug(app.Role), pdfSuffix)

	pdfDir := filepath.Join(base, app.FolderPath, "pdf")
	if err := os.MkdirAll(pdfDir, 0o755); err != nil {
		http.Error(w, fmt.Sprintf("create pdf dir: %v", err), http.StatusInternalServerError)
		return
	}

	outPath := filepath.Join(pdfDir, pdfName)
	if err := os.WriteFile(outPath, pdfBytes, 0o644); err != nil {
		http.Error(w, fmt.Sprintf("write PDF: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"filename": pdfName})
}
