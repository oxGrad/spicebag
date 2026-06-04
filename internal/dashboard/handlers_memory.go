// internal/dashboard/handlers_memory.go
package dashboard

import (
	"database/sql"
	"net/http"

	memoryPkg "github.com/oxGrad/spicebag/internal/memory"
)

func (s *Server) handleAPIMemoriesList(w http.ResponseWriter, r *http.Request) {
	memories, err := s.mem.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if memories == nil {
		memories = []*memoryPkg.Memory{}
	}
	writeJSON(w, memories)
}

func (s *Server) handleAPIMemoriesGet(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	m, err := s.mem.GetByName(name)
	if err == sql.ErrNoRows || m == nil {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, m)
}
