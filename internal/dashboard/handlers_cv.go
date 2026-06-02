// internal/dashboard/handlers_cv.go
package dashboard

import (
	"net/http"

	"github.com/oxGrad/spicebag/internal/fs"
)

func (s *Server) handleAPICVList(w http.ResponseWriter, r *http.Request) {
	files, err := fs.ListCVs(s.root)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if files == nil {
		files = []fs.FileInfo{}
	}
	writeJSON(w, files)
}
