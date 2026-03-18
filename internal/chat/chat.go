package chat

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/potacast/potacast/internal/config"
	"github.com/potacast/potacast/internal/models"
)

// Chat runs an interactive chat session.
func Chat(cfg *config.Config, modelID string) error {
	model, err := resolveModel(modelID)
	if err != nil {
		return err
	}

	baseURL := fmt.Sprintf("http://%s:%d", cfg.Host, cfg.Port)
	client := &http.Client{Timeout: 60 * time.Second}

	var messages []map[string]string
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(nil, 1024*1024)

	fmt.Fprintln(os.Stderr, "Chat with", model, "(/bye to exit, /clear, /model)")
	fmt.Fprintln(os.Stderr, "---")

	for {
		line, err := readMultiLine(scanner)
		if err != nil {
			return err
		}
		if line == "" {
			continue
		}
		if line == "/bye" || line == "/exit" || line == "/quit" {
			break
		}
		if line == "/clear" {
			messages = nil
			fmt.Fprintln(os.Stderr, "History cleared.")
			continue
		}
		if strings.HasPrefix(line, "/model") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				newModel, err := resolveModel(parts[1])
				if err != nil {
					fmt.Fprintln(os.Stderr, "Error:", err)
					continue
				}
				model = newModel
				fmt.Fprintln(os.Stderr, "Switched to", model)
			} else {
				local, _ := models.ListLocal()
				fmt.Fprintln(os.Stderr, "Available models:")
				for _, m := range local {
					marker := " "
					if m.ID == model {
						marker = "*"
					}
					fmt.Fprintf(os.Stderr, "  %s %s\n", marker, m.ID)
				}
				fmt.Fprint(os.Stderr, "Switch to (name): ")
				if scanner.Scan() {
					name := strings.TrimSpace(scanner.Text())
					if name != "" {
						newModel, err := resolveModel(name)
						if err != nil {
							fmt.Fprintln(os.Stderr, "Error:", err)
						} else {
							model = newModel
							fmt.Fprintln(os.Stderr, "Switched to", model)
						}
					}
				}
			}
			continue
		}

		messages = append(messages, map[string]string{"role": "user", "content": line})
		assistantContent, err := streamCompletion(client, baseURL, model, messages)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			messages = messages[:len(messages)-1]
			continue
		}
		messages = append(messages, map[string]string{"role": "assistant", "content": assistantContent})
		if len(messages) > 20 {
			messages = messages[len(messages)-20:]
		}
	}
	return scanner.Err()
}

// readMultiLine reads input until a non-continuation line or empty line.
// Lines ending with \ are continued.
func readMultiLine(scanner *bufio.Scanner) (string, error) {
	var lines []string
	for {
		fmt.Print(">>> ")
		if !scanner.Scan() {
			if len(lines) > 0 {
				return strings.TrimSpace(strings.Join(lines, "\n")), nil
			}
			return "", scanner.Err()
		}
		line := scanner.Text()
		trimmed := strings.TrimRight(line, " \t")
		if trimmed == "" {
			if len(lines) == 0 {
				continue
			}
			return strings.TrimSpace(strings.Join(lines, "\n")), nil
		}
		if strings.HasSuffix(trimmed, "\\") {
			lines = append(lines, strings.TrimSuffix(trimmed, "\\"))
			continue
		}
		lines = append(lines, line)
		return strings.TrimSpace(strings.Join(lines, "\n")), nil
	}
}

func resolveModel(modelID string) (string, error) {
	local, err := models.ListLocal()
	if err != nil {
		return "", err
	}
	if len(local) == 0 {
		return "", fmt.Errorf("no models downloaded. Use 'potacast pull <model-id>' first")
	}
	if modelID == "" {
		return local[0].ID, nil
	}
	dirName := models.RepoToDirName(modelID)
	repo, _ := models.ParseModelID(modelID)
	if repo != modelID {
		dirName = models.RepoToDirName(repo)
	}
	for _, m := range local {
		if m.ID == modelID || m.ID == dirName {
			return m.ID, nil
		}
	}
	return "", fmt.Errorf("model %q not found. Use 'potacast list' to see available models", modelID)
}

type chatRequest struct {
	Model    string              `json:"model"`
	Messages []map[string]string `json:"messages"`
	Stream   bool                `json:"stream"`
}

type streamChunk struct {
	Choices []struct {
		Delta struct {
			Content         string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

func streamCompletion(client *http.Client, baseURL, model string, messages []map[string]string) (string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Cancel on SIGINT (Ctrl+C)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT)
	defer signal.Stop(sigCh)
	go func() {
		<-sigCh
		cancel()
	}()

	body, err := json.Marshal(chatRequest{
		Model:    model,
		Messages: messages,
		Stream:   true,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return "", fmt.Errorf("interrupted")
		}
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		errMsg := string(b)
		err := fmt.Errorf("API error %d: %s", resp.StatusCode, errMsg)
		if (resp.StatusCode == 404 || resp.StatusCode >= 500) && (strings.Contains(errMsg, "model") || strings.Contains(strings.ToLower(errMsg), "not found")) {
			err = fmt.Errorf("%w\n\nModel may not be loaded. Try 'potacast list' to see available models, or 'potacast pull <model-id>' if not downloaded.", err)
		}
		return "", err
	}

	var sb strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(nil, 1024*1024)
	for scanner.Scan() {
		if ctx.Err() != nil {
			fmt.Println()
			return sb.String(), fmt.Errorf("interrupted")
		}
		line := scanner.Bytes()
		if !bytes.HasPrefix(line, []byte("data: ")) {
			continue
		}
		data := bytes.TrimPrefix(line, []byte("data: "))
		if bytes.Equal(data, []byte("[DONE]")) {
			break
		}
		var chunk streamChunk
		if err := json.Unmarshal(data, &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta
		if delta.ReasoningContent != "" {
			fmt.Fprint(os.Stderr, "[思考] ", delta.ReasoningContent)
			sb.WriteString(delta.ReasoningContent)
		}
		if delta.Content != "" {
			fmt.Print(delta.Content)
			sb.WriteString(delta.Content)
		}
	}
	fmt.Println()
	return sb.String(), scanner.Err()
}

// RunNonInteractive reads prompt from stdin, calls the API once, writes response to stdout.
func RunNonInteractive(cfg *config.Config, modelID string, r io.Reader, w io.Writer) error {
	model, err := resolveModel(modelID)
	if err != nil {
		return err
	}

	prompt, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	text := strings.TrimSpace(string(prompt))
	if text == "" {
		return nil
	}

	baseURL := fmt.Sprintf("http://%s:%d", cfg.Host, cfg.Port)
	client := &http.Client{Timeout: 120 * time.Second}

	messages := []map[string]string{{"role": "user", "content": text}}
	body, err := json.Marshal(chatRequest{Model: model, Messages: messages, Stream: true})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(b))
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(nil, 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if !bytes.HasPrefix(line, []byte("data: ")) {
			continue
		}
		data := bytes.TrimPrefix(line, []byte("data: "))
		if bytes.Equal(data, []byte("[DONE]")) {
			break
		}
		var chunk streamChunk
		if err := json.Unmarshal(data, &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta
			if delta.ReasoningContent != "" {
				fmt.Fprint(w, delta.ReasoningContent)
			}
			if delta.Content != "" {
				fmt.Fprint(w, delta.Content)
			}
		}
	}
	return scanner.Err()
}

// WaitForServer polls the server until it responds or timeout.
func WaitForServer(cfg *config.Config, timeout time.Duration) error {
	baseURL := fmt.Sprintf("http://%s:%d", cfg.Host, cfg.Port)
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	for time.Now().Before(deadline) {
		resp, err := client.Get(baseURL + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	return fmt.Errorf("server did not become ready within %v", timeout)
}
