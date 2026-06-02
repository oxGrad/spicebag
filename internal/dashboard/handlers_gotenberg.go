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
	Err     string `json:"error,omitempty"`
}

const gotenbergImage     = "gotenberg/gotenberg:8"
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

func (s *Server) waitForGotenberg(maxAttempts int) bool {
	for i := range maxAttempts {
		if s.checkGotenberg() {
			return true
		}
		if i < maxAttempts-1 {
			time.Sleep(time.Second)
		}
	}
	return false
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
		// stop+remove any stale container first (ignore errors)
		runCmd("docker", "stop", gotenbergContainer)  //nolint:errcheck
		runCmd("docker", "rm", gotenbergContainer)    //nolint:errcheck
		return runCmd("docker", "run", "-d", "--name", gotenbergContainer, "-p", "3000:3000", gotenbergImage)
	case "podman":
		runCmd("podman", "stop", gotenbergContainer) //nolint:errcheck
		runCmd("podman", "rm", gotenbergContainer)   //nolint:errcheck
		return runCmd("podman", "run", "-d", "--name", gotenbergContainer, "-p", "3000:3000", gotenbergImage)
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

func (s *Server) handleAPIGotenbergStart(w http.ResponseWriter, r *http.Request) {
	resp := gotenbergStatusJSON{}
	if err := s.startGotenberg(); err != nil {
		resp.Err = err.Error()
		resp.Running = s.checkGotenberg()
	} else {
		resp.Running = s.waitForGotenberg(5)
	}
	writeJSON(w, resp)
}

func (s *Server) handleAPIGotenbergStop(w http.ResponseWriter, r *http.Request) {
	resp := gotenbergStatusJSON{}
	if err := s.stopGotenberg(); err != nil {
		resp.Err = err.Error()
	}
	resp.Running = s.checkGotenberg()
	writeJSON(w, resp)
}
