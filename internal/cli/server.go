package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/potacast/potacast/internal/config"
	"github.com/potacast/potacast/internal/server"
	"github.com/spf13/cobra"
)

func newServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Manage the background server",
		Long:  "Start or stop the API server that runs in the background.",
	}
	cmd.AddCommand(newServerStartCmd())
	cmd.AddCommand(newServerStopCmd())
	cmd.AddCommand(newServerStatusCmd())
	return cmd
}

func newServerStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the server in the background",
		RunE:  runServerStart,
	}
	cmd.Flags().Int("parallel", 0, "number of concurrent slots (-1 = auto)")
	cmd.Flags().Int("ctx", 0, "context window size")
	cmd.Flags().Int("threads", 0, "CPU threads (-1 = auto)")
	cmd.Flags().Int("batch-size", 0, "logical maximum batch size")
	cmd.Flags().Int("n-predict", 0, "max tokens to generate (-1 = unlimited)")
	cmd.Flags().Int("cache-ram", 0, "KV cache size in MiB")
	cmd.Flags().Bool("embeddings", true, "enable chat and embedding models (default: true)")
	cmd.Flags().Bool("foreground", false, "run in foreground (internal use by systemd)")
	cmd.Flags().Bool("log-file", false, "write server logs to file (default on non-Linux)")
	return cmd
}

func runServerStart(cmd *cobra.Command, args []string) error {
	if server.IsRunning() {
		return fmt.Errorf("server is already running. Use 'potacast server stop' first")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Override with CLI flags when set
	if cmd.Flags().Changed("parallel") {
		cfg.Parallel, _ = cmd.Flags().GetInt("parallel")
	}
	if cmd.Flags().Changed("ctx") {
		cfg.Ctx, _ = cmd.Flags().GetInt("ctx")
	}
	if cmd.Flags().Changed("threads") {
		cfg.Threads, _ = cmd.Flags().GetInt("threads")
	}
	if cmd.Flags().Changed("batch-size") {
		cfg.BatchSize, _ = cmd.Flags().GetInt("batch-size")
	}
	if cmd.Flags().Changed("n-predict") {
		cfg.NPredict, _ = cmd.Flags().GetInt("n-predict")
	}
	if cmd.Flags().Changed("cache-ram") {
		cfg.CacheRAM, _ = cmd.Flags().GetInt("cache-ram")
	}
	if cmd.Flags().Changed("embeddings") {
		cfg.Embeddings, _ = cmd.Flags().GetBool("embeddings")
	}

	// When --foreground: run directly (used by systemd-run)
	if foreground, _ := cmd.Flags().GetBool("foreground"); foreground {
		return server.StartForeground(cfg)
	}

	// On Linux with systemd: use systemd-run so logs go to journalctl
	if runtime.GOOS == "linux" {
		if err := startViaSystemd(cmd, cfg); err == nil {
			return nil
		}
		// Fall through to background mode if systemd-run fails
	}

	useLogFile := cmd.Flags().Changed("log-file") || runtime.GOOS != "linux"
	if err := server.StartBackground(cfg, useLogFile); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Server started at http://%s:%d\n", cfg.Host, cfg.Port)
	fmt.Fprintln(os.Stderr, "OpenAI API: http://"+cfg.Host+fmt.Sprintf(":%d/v1", cfg.Port))
	return nil
}

// startViaSystemd runs the server via systemd-run so journalctl -u potacast -f works.
func startViaSystemd(cmd *cobra.Command, cfg *config.Config) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return err
	}

	// Build args for inner process: server start --foreground + any user flags
	innerArgs := []string{"server", "start", "--foreground"}
	if cmd.Flags().Changed("parallel") {
		v, _ := cmd.Flags().GetInt("parallel")
		innerArgs = append(innerArgs, "--parallel", fmt.Sprintf("%d", v))
	}
	if cmd.Flags().Changed("ctx") {
		v, _ := cmd.Flags().GetInt("ctx")
		innerArgs = append(innerArgs, "--ctx", fmt.Sprintf("%d", v))
	}
	if cmd.Flags().Changed("threads") {
		v, _ := cmd.Flags().GetInt("threads")
		innerArgs = append(innerArgs, "--threads", fmt.Sprintf("%d", v))
	}
	if cmd.Flags().Changed("batch-size") {
		v, _ := cmd.Flags().GetInt("batch-size")
		innerArgs = append(innerArgs, "--batch-size", fmt.Sprintf("%d", v))
	}
	if cmd.Flags().Changed("n-predict") {
		v, _ := cmd.Flags().GetInt("n-predict")
		innerArgs = append(innerArgs, "--n-predict", fmt.Sprintf("%d", v))
	}
	if cmd.Flags().Changed("cache-ram") {
		v, _ := cmd.Flags().GetInt("cache-ram")
		innerArgs = append(innerArgs, "--cache-ram", fmt.Sprintf("%d", v))
	}
	if cmd.Flags().Changed("embeddings") {
		v, _ := cmd.Flags().GetBool("embeddings")
		innerArgs = append(innerArgs, fmt.Sprintf("--embeddings=%v", v))
	}

	systemdArgs := append([]string{"--user", "-u", "potacast-server.service", exe}, innerArgs...)
	c := exec.Command("systemd-run", systemdArgs...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		// Clear failed state and stop any existing unit, then retry once
		exec.Command("systemctl", "--user", "reset-failed").Run()
		exec.Command("systemctl", "--user", "stop", "potacast-server.service").Run()
		exec.Command("systemctl", "--user", "daemon-reload").Run()
		c2 := exec.Command("systemd-run", systemdArgs...)
		c2.Stdout = os.Stdout
		c2.Stderr = os.Stderr
		if err2 := c2.Run(); err2 != nil {
			return err
		}
	}

	fmt.Fprintf(os.Stderr, "Server started at http://%s:%d\n", cfg.Host, cfg.Port)
	fmt.Fprintln(os.Stderr, "OpenAI API: http://"+cfg.Host+fmt.Sprintf(":%d/v1", cfg.Port))
	fmt.Fprintln(os.Stderr, "Logs: journalctl --user -u potacast-server -f")
	return nil
}

func newServerStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the server",
		RunE:  runServerStop,
	}
}

func runServerStop(cmd *cobra.Command, args []string) error {
	if !server.IsRunning() {
		return fmt.Errorf("server is not running")
	}

	if err := server.Stop(); err != nil {
		return err
	}

	fmt.Println("Server stopped.")
	return nil
}

func newServerStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show server status",
		RunE:  runServerStatus,
	}
}

func runServerStatus(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if !server.IsRunning() {
		fmt.Println("Status: stopped")
		fmt.Printf("Port: %d (from config)\n", cfg.Port)
		return nil
	}

	pid, err := server.GetPID()
	if err != nil {
		return err
	}

	fmt.Println("Status: running")
	fmt.Printf("Port: %d\n", cfg.Port)
	fmt.Printf("PID: %d\n", pid)
	fmt.Printf("API: http://%s:%d/v1\n", cfg.Host, cfg.Port)
	return nil
}
