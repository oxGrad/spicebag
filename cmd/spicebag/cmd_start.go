// cmd/spicebag/cmd_start.go
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/oxGrad/spicebag/internal/config"
	"github.com/oxGrad/spicebag/internal/dashboard"
	"github.com/oxGrad/spicebag/internal/db"
	"github.com/oxGrad/spicebag/internal/memory"
	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	var daemon bool

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the web dashboard",
		RunE: func(cmd *cobra.Command, args []string) error {
			root := spicebagRoot()
			if needsInit(root) {
				fmt.Println("First run — setting up Spice Bag...")
				if err := runInit(root, os.Stdout); err != nil {
					return fmt.Errorf("auto-init: %w", err)
				}
				fmt.Println()
			}
			pidPath := filepath.Join(root, "spicebag.pid")
			logPath := filepath.Join(root, "spicebag.log")

			if daemon {
				logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
				if err != nil {
					return fmt.Errorf("open log: %w", err)
				}

				child := exec.Command(os.Args[0], "start")
				child.Stdout = logFile
				child.Stderr = logFile
				child.SysProcAttr = &syscall.SysProcAttr{Setsid: true} // detach from terminal
				if err := child.Start(); err != nil {
					return fmt.Errorf("start background server: %w", err)
				}
				logFile.Close()

				pidStr := strconv.Itoa(child.Process.Pid) + "\n"
				if err := os.WriteFile(pidPath, []byte(pidStr), 0o644); err != nil {
					return fmt.Errorf("write PID: %w", err)
				}

				fmt.Printf("Dashboard started in background (PID %d)\n", child.Process.Pid)
				fmt.Printf("Log:  %s\n", logPath)
				fmt.Printf("Stop: spicebag stop\n")
				return nil
			}

			// foreground mode
			cfgPath := filepath.Join(root, "config.toml")
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			store, err := db.Open(filepath.Join(root, "spicebag.db"))
			if err != nil {
				return fmt.Errorf("open db: %w", err)
			}
			defer store.Close()

			mem, err := memory.Open(filepath.Join(root, "memory.db"))
			if err != nil {
				return fmt.Errorf("open memory db: %w", err)
			}
			defer mem.Close()

			addr := fmt.Sprintf(":%d", cfg.DashboardPort)
			fmt.Printf("Dashboard running at http://localhost%s\n", addr)
			return dashboard.NewServer(root, store, mem, cfg).Serve(addr)
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
			pidPath := filepath.Join(spicebagRoot(), "spicebag.pid")
			data, err := os.ReadFile(pidPath)
			if os.IsNotExist(err) {
				return fmt.Errorf("spicebag is not running (no PID file at %s)", pidPath)
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
			fmt.Printf("Stopped spicebag dashboard (PID %d)\n", pid)
			return nil
		},
	}
}
