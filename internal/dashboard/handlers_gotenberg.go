// internal/dashboard/handlers_gotenberg.go
package dashboard

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type gotenbergStatusJSON struct {
	Running bool   `json:"running"`
	Pulling bool   `json:"pulling,omitempty"`
	Err     string `json:"error,omitempty"`
}

const gotenbergImage     = "docker.io/gotenberg/gotenberg:8"
const gotenbergContainer = "spicebag-gotenberg"

func (s *Server) checkGotenberg() bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(s.cfg.GotenbergURL + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}


func runCmd(binary string, args ...string) error {
	var stderr bytes.Buffer
	cmd := exec.Command(binary, args...)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func (s *Server) defaultStartGotenberg() error {
	switch s.cfg.GotenbergRuntime {
	case "", "docker-compose":
		composePath := filepath.Join(s.root, "docker-compose.yml")
		if _, err := os.Stat(composePath); os.IsNotExist(err) {
			return fmt.Errorf("docker-compose.yml not found at %s — run spicebag init first", composePath)
		}
		return runCmd("docker", "compose", "-f", composePath, "up", "-d")
	case "podman-compose":
		composePath := filepath.Join(s.root, "docker-compose.yml")
		if _, err := os.Stat(composePath); os.IsNotExist(err) {
			return fmt.Errorf("docker-compose.yml not found at %s — run spicebag init first", composePath)
		}
		return runCmd("podman", "compose", "-f", composePath, "up", "-d")
	case "docker":
		runCmd("docker", "stop", gotenbergContainer) //nolint:errcheck
		return runCmd("docker", "run", "-d", "--rm", "--name", gotenbergContainer, "-p", "3000:3000", gotenbergImage)
	case "podman":
		runCmd("podman", "stop", gotenbergContainer) //nolint:errcheck
		return runCmd("podman", "run", "-d", "--rm", "--name", gotenbergContainer, "-p", "3000:3000", gotenbergImage)
	default:
		return fmt.Errorf("unknown gotenberg_runtime %q — valid values: docker-compose, docker, podman-compose, podman", s.cfg.GotenbergRuntime)
	}
}

func (s *Server) defaultStopGotenberg() error {
	switch s.cfg.GotenbergRuntime {
	case "", "docker-compose":
		composePath := filepath.Join(s.root, "docker-compose.yml")
		if _, err := os.Stat(composePath); os.IsNotExist(err) {
			return fmt.Errorf("docker-compose.yml not found at %s", composePath)
		}
		return runCmd("docker", "compose", "-f", composePath, "stop", "gotenberg")
	case "podman-compose":
		composePath := filepath.Join(s.root, "docker-compose.yml")
		if _, err := os.Stat(composePath); os.IsNotExist(err) {
			return fmt.Errorf("docker-compose.yml not found at %s", composePath)
		}
		return runCmd("podman", "compose", "-f", composePath, "stop", "gotenberg")
	case "docker":
		return runCmd("docker", "stop", gotenbergContainer)
	case "podman":
		return runCmd("podman", "stop", gotenbergContainer)
	default:
		return fmt.Errorf("unknown gotenberg_runtime %q", s.cfg.GotenbergRuntime)
	}
}

func (s *Server) handleAPIGotenbergStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, gotenbergStatusJSON{Running: s.checkGotenberg()})
}

// imageNeedsPull returns true if the Gotenberg image is not yet present locally.
// Only checked for direct docker/podman runtimes; compose mode returns false.
func (s *Server) imageNeedsPull() bool {
	switch s.cfg.GotenbergRuntime {
	case "docker":
		return runCmd("docker", "image", "inspect", "--format", "{{.Id}}", gotenbergImage) != nil
	case "podman", "":
		return runCmd("podman", "image", "exists", gotenbergImage) != nil
	default:
		return false
	}
}

func (s *Server) handleAPIGotenbergStart(w http.ResponseWriter, r *http.Request) {
	// Check before starting so the UI can show a "pulling" message.
	pulling := s.imageNeedsPull()
	// Run in background — image pull can block for minutes on first run.
	// Client polls /api/gotenberg/status until running.
	go s.startGotenberg() //nolint:errcheck
	writeJSON(w, gotenbergStatusJSON{Running: s.checkGotenberg(), Pulling: pulling})
}

func (s *Server) handleAPIGotenbergStop(w http.ResponseWriter, r *http.Request) {
	resp := gotenbergStatusJSON{}
	if err := s.stopGotenberg(); err != nil {
		resp.Err = err.Error()
	}
	resp.Running = s.checkGotenberg()
	writeJSON(w, resp)
}
