// internal/dashboard/handlers_gotenberg.go
package dashboard

import (
	"bytes"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type gotenbergStatusData struct {
	Running  bool
	Err      string
	FilePath string
	Theme    string
}

// SetComposeRunner replaces the docker compose executor — used in tests.
func (s *Server) SetComposeRunner(fn func(args ...string) error) {
	s.runCompose = fn
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

// waitForGotenberg retries checkGotenberg up to maxAttempts times with 1-second pauses.
func (s *Server) waitForGotenberg(maxAttempts int) bool {
	for i := 0; i < maxAttempts; i++ {
		if s.checkGotenberg() {
			return true
		}
		time.Sleep(time.Second)
	}
	return false
}

func (s *Server) defaultRunCompose(args ...string) error {
	composePath := filepath.Join(s.root, "docker-compose.yml")
	baseArgs := append([]string{"compose", "-f", composePath}, args...)
	cmd := exec.Command("docker", baseArgs...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func (s *Server) handleGotenbergStatus(w http.ResponseWriter, r *http.Request) {
	data := gotenbergStatusData{
		Running:  s.checkGotenberg(),
		FilePath: r.URL.Query().Get("file_path"),
		Theme:    r.URL.Query().Get("theme"),
	}
	s.renderPartial(w, "gotenberg_status.html", "gotenberg-status", data)
}

func (s *Server) handleGotenbergStart(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "parse form: "+err.Error(), http.StatusBadRequest)
		return
	}
	data := gotenbergStatusData{
		FilePath: r.FormValue("file_path"),
		Theme:    r.FormValue("theme"),
	}
	if err := s.runCompose("up", "-d"); err != nil {
		data.Err = err.Error()
		data.Running = s.checkGotenberg()
	} else {
		data.Running = s.waitForGotenberg(5)
	}
	s.renderPartial(w, "gotenberg_status.html", "gotenberg-status", data)
}

func (s *Server) handleGotenbergStop(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "parse form: "+err.Error(), http.StatusBadRequest)
		return
	}
	data := gotenbergStatusData{
		FilePath: r.FormValue("file_path"),
		Theme:    r.FormValue("theme"),
	}
	if err := s.runCompose("stop", "gotenberg"); err != nil {
		data.Err = err.Error()
	}
	data.Running = s.checkGotenberg()
	s.renderPartial(w, "gotenberg_status.html", "gotenberg-status", data)
}
