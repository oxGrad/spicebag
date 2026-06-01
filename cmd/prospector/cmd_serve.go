// cmd/prospector/cmd_serve.go
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/graditya/prospector/internal/config"
	"github.com/graditya/prospector/internal/dashboard"
	"github.com/graditya/prospector/internal/db"
	"github.com/spf13/cobra"
)

func newServeCmd() *cobra.Command {
	var daemon bool

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the web dashboard",
		RunE: func(cmd *cobra.Command, args []string) error {
			root := prospectorRoot()
			pidPath := filepath.Join(root, "prospector.pid")
			logPath := filepath.Join(root, "prospector.log")

			if daemon {
				logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
				if err != nil {
					return fmt.Errorf("open log: %w", err)
				}

				child := exec.Command(os.Args[0], "serve")
				child.Stdout = logFile
				child.Stderr = logFile
				child.SysProcAttr = &syscall.SysProcAttr{Setsid: true} // detach from terminal
				if err := child.Start(); err != nil {
					return fmt.Errorf("start background server: %w", err)
				}
				logFile.Close()

				pidStr := strconv.Itoa(child.Process.Pid) + "\n"
				if err := os.WriteFile(pidPath, []byte(pidStr), 0644); err != nil {
					return fmt.Errorf("write PID: %w", err)
				}

				fmt.Printf("Dashboard started in background (PID %d)\n", child.Process.Pid)
				fmt.Printf("Log:  %s\n", logPath)
				fmt.Printf("Stop: prospector stop\n")
				return nil
			}

			// foreground mode
			cfgPath := filepath.Join(root, "config.toml")
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			store, err := db.Open(filepath.Join(root, "prospector.db"))
			if err != nil {
				return fmt.Errorf("open db: %w", err)
			}
			defer store.Close()

			addr := fmt.Sprintf(":%d", cfg.DashboardPort)
			fmt.Printf("Dashboard running at http://localhost%s\n", addr)
			return dashboard.NewServer(root, store, cfg).Serve(addr)
		},
	}

	cmd.Flags().BoolVarP(&daemon, "daemon", "d", false, "Run in background")
	return cmd
}

func newStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the background dashboard server",
		RunE: func(cmd *cobra.Command, args []string) error {
			pidPath := filepath.Join(prospectorRoot(), "prospector.pid")
			data, err := os.ReadFile(pidPath)
			if os.IsNotExist(err) {
				return fmt.Errorf("prospector is not running (no PID file at %s)", pidPath)
			}
			if err != nil {
				return err
			}

			pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
			if err != nil {
				return fmt.Errorf("invalid PID in %s", pidPath)
			}

			proc, err := os.FindProcess(pid)
			if err != nil {
				return fmt.Errorf("process not found: %w", err)
			}
			if err := proc.Signal(syscall.SIGTERM); err != nil {
				return fmt.Errorf("stop process: %w", err)
			}

			os.Remove(pidPath)
			fmt.Printf("Stopped prospector dashboard (PID %d)\n", pid)
			return nil
		},
	}
}
