package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/spf13/cobra"
)

const hfQuickSearchURL = "https://huggingface.co/api/quicksearch"

func newSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search for GGUF models on Hugging Face",
		Args:  cobra.ExactArgs(1),
		RunE:  runSearch,
	}
	cmd.Flags().Int("limit", 20, "maximum number of results")
	return cmd
}

type quickSearchResponse struct {
	Models []struct {
		ID string `json:"id"`
	} `json:"models"`
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := args[0]
	limit, _ := cmd.Flags().GetInt("limit")
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	u, err := url.Parse(hfQuickSearchURL)
	if err != nil {
		return err
	}
	q := u.Query()
	q.Set("q", query)
	q.Set("type", "model")
	q.Set("library", "gguf")
	q.Set("limit", fmt.Sprintf("%d", limit))
	u.RawQuery = q.Encode()

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(u.String())
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("search failed: %s", resp.Status)
	}

	var result quickSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	if len(result.Models) == 0 {
		fmt.Println("No GGUF models found.")
		return nil
	}

	for _, m := range result.Models {
		fmt.Fprintln(os.Stdout, m.ID)
	}
	return nil
}
