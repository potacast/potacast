package cli

import (
	"fmt"
	"time"

	"github.com/potacast/potacast/internal/chat"
	"github.com/potacast/potacast/internal/config"
	"github.com/potacast/potacast/internal/server"
	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run [model]",
		Short: "Start interactive chat in the terminal",
		Long:  "Open an interactive chat session. Starts the server in the background if not already running.",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runRun,
	}
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
		if err := server.StartBackground(cfg); err != nil {
			return err
		}
		if err := chat.WaitForServer(cfg, 30*time.Second); err != nil {
			return fmt.Errorf("server failed to start: %w", err)
		}
	}

	return chat.Chat(cfg, modelID)
}
