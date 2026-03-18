package cli

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/potacast/potacast/internal/models"
	"github.com/potacast/potacast/internal/paths"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List downloaded models",
		RunE:  runList,
	}
	cmd.Flags().String("sort", "name", "sort by: name, size, mtime")
	return cmd
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

	sortBy, _ := cmd.Flags().GetString("sort")
	switch sortBy {
	case "name":
		sort.Slice(localModels, func(i, j int) bool { return localModels[i].ID < localModels[j].ID })
	case "size":
		sort.Slice(localModels, func(i, j int) bool { return localModels[i].Size > localModels[j].Size })
	case "mtime":
		sort.Slice(localModels, func(i, j int) bool { return localModels[i].Mtime.After(localModels[j].Mtime) })
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tSIZE")
	fmt.Fprintln(tw, "----\t----")
	for _, m := range localModels {
		fmt.Fprintf(tw, "%s\t%s\n", m.ID, formatSize(m.Size))
	}
	return tw.Flush()
}

func formatSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for n >= div {
		n /= div
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n), "KMGTPE"[exp])
}
