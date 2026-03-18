package cli

import (
	"fmt"

	"github.com/potacast/potacast/internal/server"
	"github.com/spf13/cobra"
)

func newStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the llama-server",
		RunE:  runStop,
	}
}

func runStop(cmd *cobra.Command, args []string) error {
	if !server.IsRunning() {
		return fmt.Errorf("server is not running")
	}

	if err := server.Stop(); err != nil {
		return err
	}

	fmt.Println("Server stopped.")
	return nil
}
