package cli

import (
	"fmt"
	"os"

	"github.com/potacast/potacast/internal/config"
	"github.com/potacast/potacast/internal/server"
	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run [model]",
		Short: "Start the llama-server (router mode)",
		Long:  "Start llama-server in router mode. Optionally specify a model to preload.",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runRun,
	}
}

func runRun(cmd *cobra.Command, args []string) error {
	if server.IsRunning() {
		return fmt.Errorf("server is already running. Use 'potacast stop' first")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	_ = args // model preload not implemented in v1 - router auto-loads on request

	fmt.Fprintf(os.Stderr, "Starting server at http://%s:%d\n", cfg.Host, cfg.Port)
	fmt.Fprintln(os.Stderr, "OpenAI API: http://"+cfg.Host+fmt.Sprintf(":%d/v1", cfg.Port))

	return server.Start(cfg)
}
