package cli

import (
	"fmt"
	"os"

	"github.com/potacast/potacast/internal/config"
	"github.com/potacast/potacast/internal/paths"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
	}
	cmd.AddCommand(newConfigInitCmd())
	cmd.AddCommand(newConfigPathCmd())
	return cmd
}

func newConfigInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create default config file if it does not exist",
		RunE:  runConfigInit,
	}
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	fpath := paths.ConfigFile()
	if _, err := os.Stat(fpath); err == nil {
		fmt.Fprintf(os.Stderr, "Config already exists at %s\n", fpath)
		return nil
	}
	cfg := config.Default()
	if err := cfg.Save(); err != nil {
		return err
	}
	fmt.Printf("Created config at %s\n", fpath)
	return nil
}

func newConfigPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print config file path",
		RunE:  runConfigPath,
	}
}

func runConfigPath(cmd *cobra.Command, args []string) error {
	fmt.Println(paths.ConfigFile())
	return nil
}
