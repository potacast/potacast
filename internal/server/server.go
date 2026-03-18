package server

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/potacast/potacast/internal/config"
	"github.com/potacast/potacast/internal/paths"
)

const (
	llamaRepo     = "ggml-org/llama.cpp"
	githubAPIBase = "https://api.github.com/repos"
)

type githubRelease struct {
	TagName string         `json:"tag_name"`
	Assets  []githubAsset  `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// pickAsset selects the best llama.cpp release asset for the current platform.
// Prefers CPU build for simplicity (v1).
func pickAsset(assets []githubAsset) (string, error) {
	osName := "linux"
	if runtime.GOOS == "darwin" {
		osName = "macos"
	}

	arch := runtime.GOARCH
	archSuffix := arch
	if arch == "amd64" {
		archSuffix = "x64"
	} else if arch == "arm64" {
		archSuffix = "aarch64" // llama.cpp uses aarch64 in asset names
	}

	var candidates []string
	for _, a := range assets {
		name := strings.ToLower(a.Name)
		if !strings.HasPrefix(name, "llama-") || !strings.Contains(name, "-bin-") {
			continue
		}
		if strings.Contains(name, "win-") || strings.Contains(name, "windows") {
			continue
		}
		if osName == "linux" {
			if !strings.Contains(name, "ubuntu") && !strings.Contains(name, "openeuler") && !strings.Contains(name, "linux") {
				continue
			}
			// Prefer CPU-only for v1 (avoid cuda, rocm, vulkan - they need extra libs)
			// Note: ubuntu-x64 may need libopenvino; openEuler may need libascendcl on some systems
			if strings.Contains(name, "cuda") || strings.Contains(name, "rocm") || strings.Contains(name, "vulkan") {
				continue
			}
			// Prefer ubuntu-x64 (most common) over openEuler
			if strings.Contains(name, "ubuntu") && !strings.Contains(name, "openvino") {
				candidates = append([]string{a.BrowserDownloadURL}, candidates...)
				continue
			}
		}
		if osName == "macos" {
			if !strings.Contains(name, "macos") {
				continue
			}
		}
		if strings.Contains(name, archSuffix) || (archSuffix == "x64" && strings.Contains(name, "x86")) {
			candidates = append(candidates, a.BrowserDownloadURL)
		}
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no compatible llama.cpp build found for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	return candidates[0], nil
}

// findLlamaServer searches for llama-server in the given directory.
func findLlamaServer(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		path := filepath.Join(dir, e.Name())
		if e.IsDir() {
			if found, err := findLlamaServer(path); err == nil && found != "" {
				return found, nil
			}
		} else if e.Name() == "llama-server" {
			return path, nil
		}
	}
	return "", fmt.Errorf("llama-server not found")
}

// EnsureLlamaServer downloads llama-server if not present.
// Extracts the full tarball so shared libs (libmtmd.so, etc.) are available.
func EnsureLlamaServer() (string, error) {
	binDir := paths.LlamaBinDir()
	if err := paths.EnsureDir(binDir); err != nil {
		return "", err
	}

	// Check if already extracted
	if binPath, err := findLlamaServer(binDir); err == nil {
		return binPath, nil
	}

	// Fetch latest release
	url := fmt.Sprintf("%s/%s/releases/latest", githubAPIBase, llamaRepo)
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("fetch llama.cpp release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github API error: %s", resp.Status)
	}

	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", err
	}

	downloadURL, err := pickAsset(rel.Assets)
	if err != nil {
		return "", err
	}

	// Download tarball
	fmt.Fprintln(os.Stderr, "Downloading llama-server...")
	dlResp, err := http.Get(downloadURL)
	if err != nil {
		return "", err
	}
	defer dlResp.Body.Close()

	if dlResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed: %s", dlResp.Status)
	}

	gzr, err := gzip.NewReader(dlResp.Body)
	if err != nil {
		return "", err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		destPath := filepath.Join(binDir, h.Name)
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return "", err
			}
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return "", err
			}
			if err := os.Symlink(h.Linkname, destPath); err != nil {
				return "", err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return "", err
			}
			f, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return "", err
			}
			f.Close()
			os.Chmod(destPath, 0755)
		}
	}

	binPath, err := findLlamaServer(binDir)
	if err != nil {
		return "", fmt.Errorf("llama-server not found after extraction: %w", err)
	}
	fmt.Fprintln(os.Stderr, "llama-server ready at", binPath)
	return binPath, nil
}

// Start launches llama-server in router mode.
func Start(cfg *config.Config) error {
	binPath, err := EnsureLlamaServer()
	if err != nil {
		return err
	}

	modelsDir := paths.ModelsDir()
	if err := paths.EnsureDir(modelsDir); err != nil {
		return err
	}

	args := []string{
		"--models-dir", modelsDir,
		"--host", cfg.Host,
		"--port", fmt.Sprintf("%d", cfg.Port),
		"--ctx-size", fmt.Sprintf("%d", cfg.Ctx),
	}

	binDir := filepath.Dir(binPath)
	cmd := exec.Command(binPath, args...)
	cmd.Dir = binDir
	env := os.Environ()
	if runtime.GOOS == "darwin" {
		env = append(env, "DYLD_LIBRARY_PATH="+binDir)
	} else {
		env = append(env, "LD_LIBRARY_PATH="+binDir)
	}
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		return err
	}

	pidFile := paths.PIDFile()
	if err := os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", cmd.Process.Pid)), 0644); err != nil {
		cmd.Process.Kill()
		return err
	}

	return cmd.Wait()
}

// Stop kills the running llama-server.
func Stop() error {
	pidFile := paths.PIDFile()
	data, err := os.ReadFile(pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("server not running (no PID file)")
		}
		return err
	}

	var pid int
	if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil {
		return fmt.Errorf("invalid PID file: %w", err)
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	if err := proc.Kill(); err != nil {
		return err
	}

	os.Remove(pidFile)
	return nil
}

// IsRunning returns true if the server appears to be running.
func IsRunning() bool {
	pidFile := paths.PIDFile()
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return false
	}
	var pid int
	if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil {
		return false
	}
	// Signal 0 checks process existence without killing
	return syscall.Kill(pid, 0) == nil
}
