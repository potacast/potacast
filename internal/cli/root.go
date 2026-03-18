package cli

import (
	"os"

	"github.com/spf13/cobra"
)

// Execute runs the root command with the given version.
func Execute(v string) {
	root := &cobra.Command{
		Use:   "potacast",
		Short: "Run GGUF models locally with llama.cpp",
		Long:  "Potacast: Download and run GGUF models from Hugging Face with OpenAI-compatible API.",
	}

	root.Version = v

	root.AddCommand(newPullCmd())
	root.AddCommand(newListCmd())
	root.AddCommand(newRunCmd())
	root.AddCommand(newStopCmd())
	root.AddCommand(newRmCmd())

	serveCmd := newRunCmd()
	serveCmd.Use = "serve"
	serveCmd.Short = "Start the server (alias for run)"
	root.AddCommand(serveCmd)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
