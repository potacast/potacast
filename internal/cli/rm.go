package cli

import (
	"fmt"

	"github.com/potacast/potacast/internal/models"
	"github.com/spf13/cobra"
)

func newRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm [model]",
		Short: "Remove a downloaded model",
		Args:  cobra.ExactArgs(1),
		RunE:  runRm,
	}
}

func runRm(cmd *cobra.Command, args []string) error {
	model := args[0]

	if err := models.Remove(model); err != nil {
		return err
	}

	fmt.Printf("Removed %s\n", model)
	return nil
}
