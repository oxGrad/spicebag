// internal/dashboard/handlers_gotenberg.go
package dashboard

import (
	"net/http"
	"time"
)

type gotenbergStatusJSON struct {
	Running bool `json:"running"`
}

func (s *Server) checkGotenberg() bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(s.cfg.GotenbergURL + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (s *Server) handleAPIGotenbergStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, gotenbergStatusJSON{Running: s.checkGotenberg()})
}
