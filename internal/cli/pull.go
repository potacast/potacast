package cli

import (
	"fmt"
	"os"

	"github.com/potacast/potacast/internal/models"
	"github.com/potacast/potacast/internal/paths"
	"github.com/spf13/cobra"
)

func newPullCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pull [model-id]",
		Short: "Download a GGUF model from Hugging Face",
		Long:  "Download a GGUF model. Format: org/model, org/model:Q4_K_M, org/model:branch, or org/model:branch:Q4_K_M",
		Args:  cobra.ExactArgs(1),
		RunE:  runPull,
	}
}

func runPull(cmd *cobra.Command, args []string) error {
	modelID := args[0]

	if err := paths.EnsureDir(paths.ModelsDir()); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Pulling %s...\n", modelID)

	progress := func(written, total int64) {
		if total > 0 {
			pct := float64(written) / float64(total) * 100
			fmt.Fprintf(os.Stderr, "\r  %.1f%% (%d / %d MB)", pct, written/(1024*1024), total/(1024*1024))
		} else {
			fmt.Fprintf(os.Stderr, "\r  %d MB downloaded", written/(1024*1024))
		}
	}

	if err := models.Pull(modelID, progress); err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr, "\nDone.")
	return nil
}
