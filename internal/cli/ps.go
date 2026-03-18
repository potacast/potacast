package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"text/tabwriter"
	"time"

	"github.com/potacast/potacast/internal/config"
	"github.com/potacast/potacast/internal/server"
	"github.com/spf13/cobra"
)

func newPsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ps",
		Short: "List models currently loaded in the server",
		Long:  "Show models that are currently loaded and running (similar to ollama ps).",
		RunE:  runPs,
	}
}

type modelsResponse struct {
	Data []modelEntry `json:"data"`
}

type modelEntry struct {
	ID     string       `json:"id"`
	Path   string       `json:"path,omitempty"`
	Status modelStatus  `json:"status"`
	Size   int64        `json:"size,omitempty"`
}

type modelStatus struct {
	Value string `json:"value"`
}

func runPs(cmd *cobra.Command, args []string) error {
	if !server.IsRunning() {
		fmt.Fprintln(os.Stderr, "Server is not running. Start it with 'potacast server start'.")
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	baseURL := fmt.Sprintf("http://%s:%d", cfg.Host, cfg.Port)
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(baseURL + "/models")
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}

	var result modelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Filter to only loaded models
	var loaded []modelEntry
	for _, m := range result.Data {
		if m.Status.Value == "loaded" {
			loaded = append(loaded, m)
		}
	}

	if len(loaded) == 0 {
		fmt.Println("No models currently loaded.")
		return nil
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tSIZE\tSTATUS")
	fmt.Fprintln(tw, "----\t----\t------")
	for _, m := range loaded {
		sizeStr := "-"
		if m.Size > 0 {
			sizeStr = formatSize(m.Size)
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\n", m.ID, sizeStr, m.Status.Value)
	}
	return tw.Flush()
}
