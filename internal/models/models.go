package models

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	hfResolveFmt = "%s/%s/resolve/%s/%s" // base, repo, branch, path
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
	repo, _, quant = ParseModelIDWithBranch(modelID)
	return repo, quant
}

// ParseModelIDWithBranch splits model-id into repo, branch, and quant.
// Examples:
//   - "org/model" -> repo, "main", ""
//   - "org/model:Q4_K_M" -> repo, "main", "Q4_K_M"
//   - "org/model:dev" -> repo, "dev", ""
//   - "org/model:dev:Q4_K_M" -> repo, "dev", "Q4_K_M"
func ParseModelIDWithBranch(modelID string) (repo, branch, quant string) {
	parts := strings.Split(modelID, ":")
	repo = parts[0]
	branch = "main"
	quant = ""

	if len(parts) == 2 {
		if looksLikeQuant(parts[1]) {
			quant = parts[1]
		} else {
			branch = parts[1]
		}
	} else if len(parts) >= 3 {
		branch = parts[1]
		quant = parts[2]
	}
	return repo, branch, quant
}

func looksLikeQuant(s string) bool {
	if strings.HasSuffix(strings.ToLower(s), ".gguf") {
		return true
	}
	// Q4_K_M, Q5_K_S, Q8_0, etc.
	if len(s) >= 2 && (s[0] == 'Q' || s[0] == 'q') && (s[1] >= '0' && s[1] <= '9') {
		return true
	}
	return false
}

// ListHFFiles fetches the file tree from Hugging Face API.
func ListHFFiles(repo, branch string) ([]HFFile, error) {
	if branch == "" {
		branch = "main"
	}
	url := fmt.Sprintf("%s/api/models/%s/tree/%s", hfAPIBase, repo, branch)

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
// Supports resume (Range header) and skips if file exists with correct size.
// expectedSize is optional; if > 0 and file exists with that size, skip without HEAD request.
func DownloadFile(repo, branch, filePath, destPath string, expectedSize int64, progress func(written int64, total int64)) error {
	if branch == "" {
		branch = "main"
	}
	url := fmt.Sprintf(hfResolveFmt, hfAPIBase, repo, branch, filePath)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	if token := os.Getenv("HF_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	var existingSize int64
	if fi, err := os.Stat(destPath); err == nil && fi.Mode().IsRegular() {
		existingSize = fi.Size()
		if expectedSize > 0 && existingSize == expectedSize {
			// File complete (known size from API), skip download
			if progress != nil {
				progress(existingSize, existingSize)
			}
			return nil
		}
		if existingSize > 0 {
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-", existingSize))
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var out *os.File
	var startOffset int64
	if resp.StatusCode == http.StatusPartialContent {
		// Resume - append to existing file
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}
		out, err = os.OpenFile(destPath, os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return err
		}
		defer out.Close()
		startOffset, _ = out.Seek(0, io.SeekEnd)
	} else if resp.StatusCode == http.StatusOK {
		// Full download
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}
		out, err = os.Create(destPath)
		if err != nil {
			return err
		}
		defer out.Close()
	} else {
		return fmt.Errorf("download failed: %s", resp.Status)
	}

	total := resp.ContentLength
	if total > 0 {
		total += startOffset
	}
	var dst io.Writer = out
	if progress != nil {
		dst = &progressWriter{w: out, total: total, written: startOffset, progress: progress}
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
	repo, branch, quant := ParseModelIDWithBranch(modelID)

	files, err := ListHFFiles(repo, branch)
	if err != nil {
		return err
	}

	ggufPath, err := SelectGGUFFile(files, quant)
	if err != nil {
		return err
	}

	// Get expected size from file list for skip check
	var expectedSize int64
	for _, f := range files {
		if f.Path == ggufPath {
			expectedSize = f.Size
			break
		}
	}

	dirName := RepoToDirName(repo)
	modelsDir := paths.ModelsDir()
	destDir := filepath.Join(modelsDir, dirName)
	destPath := filepath.Join(destDir, filepath.Base(ggufPath))

	if err := paths.EnsureDir(destDir); err != nil {
		return err
	}

	return DownloadFile(repo, branch, ggufPath, destPath, expectedSize, progress)
}

// LocalModel represents a model directory.
type LocalModel struct {
	ID    string    // e.g. "bartowski_Llama-3.2-1B-Instruct-GGUF"
	Path  string    // full path to directory
	Size  int64     // total size of .gguf files in bytes
	Mtime time.Time // modification time of newest .gguf file
}

// ListLocal returns all downloaded models with size and mtime.
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
		sub, err := os.ReadDir(dirPath)
		if err != nil {
			continue
		}
		var size int64
		var mtime time.Time
		hasGGUF := false
		for _, s := range sub {
			if !strings.HasSuffix(strings.ToLower(s.Name()), ".gguf") {
				continue
			}
			hasGGUF = true
			info, err := s.Info()
			if err != nil {
				continue
			}
			size += info.Size()
			if info.ModTime().After(mtime) {
				mtime = info.ModTime()
			}
		}
		if hasGGUF {
			models = append(models, LocalModel{ID: e.Name(), Path: dirPath, Size: size, Mtime: mtime})
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
