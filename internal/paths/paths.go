package paths

import (
	"os"
	"path/filepath"
)

// BaseDir returns the potacast data directory.
// Uses XDG_DATA_HOME or ~/.local/share/potacast
func BaseDir() string {
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		return filepath.Join(d, "potacast")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "potacast")
}

// ConfigDir returns the potacast config directory.
// Uses XDG_CONFIG_HOME or ~/.config/potacast
func ConfigDir() string {
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return filepath.Join(d, "potacast")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "potacast")
}

// ModelsDir returns the directory for downloaded GGUF models.
func ModelsDir() string {
	return filepath.Join(BaseDir(), "models")
}

// LlamaBinDir returns the directory for the llama-server binary.
func LlamaBinDir() string {
	return filepath.Join(BaseDir(), "llama-bin")
}

// LlamaServerPath returns the path to the llama-server executable.
func LlamaServerPath() string {
	return filepath.Join(LlamaBinDir(), "bin", "llama-server")
}

// PIDFile returns the path to the llama-server PID file.
func PIDFile() string {
	return filepath.Join(BaseDir(), "llama-server.pid")
}

// LogFile returns the path to the llama-server log file (for non-Linux background mode).
func LogFile() string {
	return filepath.Join(BaseDir(), "logs", "llama-server.log")
}

// ExportsDir returns the directory for exported model archives.
func ExportsDir() string {
	return filepath.Join(BaseDir(), "exports")
}

// ConfigFile returns the path to the config file.
func ConfigFile() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

// EnsureDir creates the directory if it does not exist.
func EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0755)
}
