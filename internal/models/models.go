package models

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/potacast/potacast/internal/paths"
)

// progressWriter wraps io.Writer to report progress.
type progressWriter struct {
	w        io.Writer
	written  int64
	total    int64
	progress func(written, total int64)
}

func (p *progressWriter) Write(b []byte) (int, error) {
	n, err := p.w.Write(b)
	if n > 0 {
		p.written += int64(n)
		if p.progress != nil {
			p.progress(p.written, p.total)
		}
	}
	return n, err
}

const (
	hfAPIBase   = "https://huggingface.co"
	hfResolveFmt = "%s/%s/resolve/main/%s"
)

// HFFile represents a file in the Hugging Face tree API response.
type HFFile struct {
	Type string `json:"type"`
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// ParseModelID splits model-id into repo and optional quant/filename.
// Examples:
//   - "org/model" -> repo="org/model", quant=""
//   - "org/model:Q4_K_M" -> repo="org/model", quant="Q4_K_M"
//   - "org/model:file.gguf" -> repo="org/model", quant="file.gguf"
func ParseModelID(modelID string) (repo, quant string) {
	if idx := strings.LastIndex(modelID, ":"); idx >= 0 {
		return modelID[:idx], modelID[idx+1:]
	}
	return modelID, ""
}

// ListHFFiles fetches the file tree from Hugging Face API.
func ListHFFiles(repo string) ([]HFFile, error) {
	url := fmt.Sprintf("%s/api/models/%s/tree/main", hfAPIBase, repo)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	if token := os.Getenv("HF_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HF API error: %s", resp.Status)
	}

	var files []HFFile
	if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
		return nil, err
	}

	return files, nil
}

// SelectGGUFFile picks the best GGUF file from the list based on quant hint.
// quant can be: "" (default Q4_K_M), "Q4_K_M", or exact filename like "model.gguf"
func SelectGGUFFile(files []HFFile, quant string) (string, error) {
	var ggufFiles []HFFile
	for _, f := range files {
		if f.Type == "file" && strings.HasSuffix(strings.ToLower(f.Path), ".gguf") {
			ggufFiles = append(ggufFiles, f)
		}
	}

	if len(ggufFiles) == 0 {
		return "", fmt.Errorf("no GGUF files found in repo")
	}

	if quant == "" {
		// Default: prefer Q4_K_M
		for _, f := range ggufFiles {
			if strings.Contains(f.Path, "Q4_K_M") {
				return f.Path, nil
			}
		}
		return ggufFiles[0].Path, nil
	}

	// Exact filename match
	if strings.HasSuffix(quant, ".gguf") {
		for _, f := range ggufFiles {
			if f.Path == quant || filepath.Base(f.Path) == quant {
				return f.Path, nil
			}
		}
		return "", fmt.Errorf("file %q not found", quant)
	}

	// Quant shorthand: Q4_K_M, Q5_K_S, etc.
	upper := strings.ToUpper(quant)
	for _, f := range ggufFiles {
		if strings.Contains(strings.ToUpper(f.Path), upper) {
			return f.Path, nil
		}
	}
	return "", fmt.Errorf("no GGUF matching %q found", quant)
}

// DownloadFile downloads a file from Hugging Face to the given path.
func DownloadFile(repo, filePath, destPath string, progress func(written int64, total int64)) error {
	url := fmt.Sprintf(hfResolveFmt, hfAPIBase, repo, filePath)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	if token := os.Getenv("HF_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s", resp.Status)
	}

	total := resp.ContentLength
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	var dst io.Writer = out
	if progress != nil {
		dst = &progressWriter{w: out, total: total, progress: progress}
	}
	_, err = io.Copy(dst, resp.Body)
	return err
}

// RepoToDirName converts repo id to a safe directory name.
func RepoToDirName(repo string) string {
	return strings.ReplaceAll(repo, "/", "_")
}

// Pull downloads a model from Hugging Face.
func Pull(modelID string, progress func(written, total int64)) error {
	repo, quant := ParseModelID(modelID)

	files, err := ListHFFiles(repo)
	if err != nil {
		return err
	}

	ggufPath, err := SelectGGUFFile(files, quant)
	if err != nil {
		return err
	}

	dirName := RepoToDirName(repo)
	modelsDir := paths.ModelsDir()
	destDir := filepath.Join(modelsDir, dirName)
	destPath := filepath.Join(destDir, filepath.Base(ggufPath))

	if err := paths.EnsureDir(destDir); err != nil {
		return err
	}

	return DownloadFile(repo, ggufPath, destPath, progress)
}

// LocalModel represents a model directory.
type LocalModel struct {
	ID   string // e.g. "bartowski_Llama-3.2-1B-Instruct-GGUF"
	Path string // full path to directory
}

// ListLocal returns all downloaded models.
func ListLocal() ([]LocalModel, error) {
	modelsDir := paths.ModelsDir()
	if err := paths.EnsureDir(modelsDir); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(modelsDir)
	if err != nil {
		return nil, err
	}

	var models []LocalModel
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dirPath := filepath.Join(modelsDir, e.Name())
		// Check if it contains at least one .gguf file
		sub, err := os.ReadDir(dirPath)
		if err != nil {
			continue
		}
		hasGGUF := false
		for _, s := range sub {
			if strings.HasSuffix(strings.ToLower(s.Name()), ".gguf") {
				hasGGUF = true
				break
			}
		}
		if hasGGUF {
			models = append(models, LocalModel{ID: e.Name(), Path: dirPath})
		}
	}

	return models, nil
}

// Remove deletes a model directory.
func Remove(modelID string) error {
	// modelID can be dir name (e.g. bartowski_Llama-3.2-1B-Instruct-GGUF) or org/model
	dirName := RepoToDirName(modelID)
	path := filepath.Join(paths.ModelsDir(), dirName)

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("model %q not found", modelID)
		}
		return err
	}

	return os.RemoveAll(path)
}
