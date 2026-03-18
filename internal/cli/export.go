package cli

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/potacast/potacast/internal/models"
	"github.com/potacast/potacast/internal/paths"
	"github.com/spf13/cobra"
)

func newExportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export [model]",
		Short: "Export model to a single archive file",
		Long:  "Pack the model directory into a .tar.gz file in ~/.local/share/potacast/exports/",
		Args:  cobra.ExactArgs(1),
		RunE:  runExport,
	}
}

func runExport(cmd *cobra.Command, args []string) error {
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
		return fmt.Errorf("model %q not found", modelID)
	}

	exportsDir := paths.ExportsDir()
	if err := paths.EnsureDir(exportsDir); err != nil {
		return err
	}

	archiveName := strings.ReplaceAll(match.ID, "/", "_") + ".tar.gz"
	archivePath := filepath.Join(exportsDir, archiveName)

	f, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	if err := filepath.Walk(match.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(match.Path, path)
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.Join(match.ID, rel)
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			src, err := os.Open(path)
			if err != nil {
				return err
			}
			_, err = io.Copy(tw, src)
			src.Close()
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	fmt.Printf("Exported to %s\n", archivePath)
	return nil
}
