package cli

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/potacast/potacast/internal/paths"
	"github.com/spf13/cobra"
)

func newImportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import [file]",
		Short: "Import model from an archive file",
		Long:  "Extract a .tar.gz or .zip archive to the models directory. Archive should contain model_name/model.gguf structure.",
		Args:  cobra.ExactArgs(1),
		RunE:  runImport,
	}
}

func runImport(cmd *cobra.Command, args []string) error {
	archivePath := args[0]

	modelsDir := paths.ModelsDir()
	if err := paths.EnsureDir(modelsDir); err != nil {
		return err
	}

	if strings.HasSuffix(strings.ToLower(archivePath), ".zip") {
		return importZip(archivePath, modelsDir)
	}
	return importTarGz(archivePath, modelsDir)
}

func importTarGz(archivePath, modelsDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		destPath := filepath.Join(modelsDir, h.Name)
		if h.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return err
			}
			continue
		}
		if h.Typeflag == tar.TypeReg {
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return err
			}
			out, err := os.Create(destPath)
			if err != nil {
				return err
			}
			_, err = io.Copy(out, tr)
			out.Close()
			if err != nil {
				return err
			}
		}
	}
	fmt.Printf("Imported from %s\n", archivePath)
	return nil
}

func importZip(archivePath, modelsDir string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		destPath := filepath.Join(modelsDir, f.Name)
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(destPath)
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		out.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	fmt.Printf("Imported from %s\n", archivePath)
	return nil
}
