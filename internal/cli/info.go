package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/potacast/potacast/internal/models"
	"github.com/spf13/cobra"
)

func newInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info [model]",
		Short: "Show model information",
		Args:  cobra.ExactArgs(1),
		RunE:  runInfo,
	}
}

func runInfo(cmd *cobra.Command, args []string) error {
	modelID := args[0]

	local, err := models.ListLocal()
	if err != nil {
		return err
	}

	dirName := models.RepoToDirName(modelID)
	repo, _ := models.ParseModelID(modelID)
	if repo != modelID {
		dirName = models.RepoToDirName(repo)
	}
	var match *models.LocalModel
	for i := range local {
		if local[i].ID == modelID || local[i].ID == dirName {
			match = &local[i]
			break
		}
	}
	if match == nil {
		return fmt.Errorf("model %q not found. Use 'potacast list' to see available models", modelID)
	}

	// Gather .gguf files and infer quant from filename
	entries, err := os.ReadDir(match.Path)
	if err != nil {
		return err
	}

	fmt.Printf("Name: %s\n", match.ID)
	fmt.Printf("Path: %s\n", match.Path)
	fmt.Printf("Size: %s\n", formatSize(match.Size))
	fmt.Println("Files:")

	for _, e := range entries {
		if !strings.HasSuffix(strings.ToLower(e.Name()), ".gguf") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		quant := inferQuant(e.Name())
		if quant != "" {
			fmt.Printf("  - %s (%s, %s)\n", e.Name(), quant, formatSize(info.Size()))
		} else {
			fmt.Printf("  - %s (%s)\n", e.Name(), formatSize(info.Size()))
		}
	}

	return nil
}

func inferQuant(filename string) string {
	upper := strings.ToUpper(filename)
	// Common patterns: Q4_K_M, Q5_K_S, Q8_0, etc.
	for _, q := range []string{"Q4_K_M", "Q4_K_S", "Q5_K_M", "Q5_K_S", "Q8_0", "Q6_K", "IQ4_NL"} {
		if strings.Contains(upper, q) {
			return q
		}
	}
	return ""
}
