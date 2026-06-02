// internal/dashboard/handlers_cl.go
package dashboard

import (
	"net/http"

	"github.com/oxGrad/spicebag/internal/fs"
)

func (s *Server) handleAPICLList(w http.ResponseWriter, r *http.Request) {
	files, err := fs.ListCoverLetters(s.root)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if files == nil {
		files = []fs.FileInfo{}
	}
	writeJSON(w, files)
}
