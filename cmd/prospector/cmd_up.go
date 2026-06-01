// cmd/prospector/cmd_up.go
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newUpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "up",
		Short: "Start Gotenberg via docker compose",
		RunE: func(cmd *cobra.Command, args []string) error {
			composePath := filepath.Join(prospectorRoot(), "docker-compose.yml")
			if _, err := os.Stat(composePath); os.IsNotExist(err) {
				return fmt.Errorf("docker-compose.yml not found at %s — run prospector init first", composePath)
			}
			c := exec.Command("docker", "compose", "-f", composePath, "up", "-d")
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		},
	}
}
