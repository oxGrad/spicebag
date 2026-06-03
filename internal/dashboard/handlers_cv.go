// internal/dashboard/handlers_cv.go
package dashboard

import (
	"net/http"

	"github.com/oxGrad/spicebag/internal/db"
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

func (s *Server) handleAPICVUsages(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	apps, err := s.store.ListApplicationsByBaseCV(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if apps == nil {
		apps = []db.ApplicationWithStatus{}
	}
	writeJSON(w, apps)
}
