package cli

import (
	"fmt"

	"github.com/potacast/potacast/internal/models"
	"github.com/potacast/potacast/internal/paths"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List downloaded models",
		RunE:  runList,
	}
}

func runList(cmd *cobra.Command, args []string) error {
	if err := paths.EnsureDir(paths.ModelsDir()); err != nil {
		return err
	}

	localModels, err := models.ListLocal()
	if err != nil {
		return err
	}

	if len(localModels) == 0 {
		fmt.Println("No models downloaded. Use 'potacast pull <model-id>' to download.")
		return nil
	}

	fmt.Println("NAME")
	fmt.Println("----")
	for _, m := range localModels {
		fmt.Println(m.ID)
	}
	return nil
}
