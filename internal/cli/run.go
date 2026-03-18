package cli

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/potacast/potacast/internal/chat"
	"github.com/potacast/potacast/internal/config"
	"github.com/potacast/potacast/internal/server"
	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [model]",
		Short: "Start interactive chat in the terminal",
		Long:  "Open an interactive chat session. Starts the server in the background if not already running.",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runRun,
	}
	cmd.Flags().Bool("no-interactive", false, "read prompt from stdin, output to stdout (e.g. echo 'Hello' | potacast run --no-interactive)")
	return cmd
}

func runRun(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	modelID := ""
	if len(args) > 0 {
		modelID = args[0]
	}

	if !server.IsRunning() {
		useLogFile := runtime.GOOS != "linux"
		if err := server.StartBackground(cfg, useLogFile); err != nil {
			return err
		}
		if err := chat.WaitForServer(cfg, 30*time.Second); err != nil {
			return fmt.Errorf("server failed to start: %w\n\nSuggestions:\n  - Run 'potacast server start' manually and check for errors\n  - Ensure port %d is not in use: 'potacast server status'\n  - Check firewall allows connections to %s:%d", err, cfg.Port, cfg.Host, cfg.Port)
		}
	}

	noInteractive, _ := cmd.Flags().GetBool("no-interactive")
	if noInteractive {
		return chat.RunNonInteractive(cfg, modelID, bufio.NewReader(os.Stdin), os.Stdout)
	}
	return chat.Chat(cfg, modelID)
}
