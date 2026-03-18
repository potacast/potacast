package chat

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
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

	fmt.Fprintln(os.Stderr, "Chat with", model, "(/bye to exit)")
	fmt.Fprintln(os.Stderr, "---")

	for {
		fmt.Print(">>> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "/bye" || line == "/exit" || line == "/quit" {
			break
		}

		messages = append(messages, map[string]string{"role": "user", "content": line})
		assistantContent, err := streamCompletion(client, baseURL, model, messages)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			messages = messages[:len(messages)-1]
			continue
		}
		messages = append(messages, map[string]string{"role": "assistant", "content": assistantContent})
		// Keep last 20 messages to avoid unbounded context
		if len(messages) > 20 {
			messages = messages[len(messages)-20:]
		}
	}
	return scanner.Err()
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
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

func streamCompletion(client *http.Client, baseURL, model string, messages []map[string]string) (string, error) {
	body, err := json.Marshal(chatRequest{
		Model:    model,
		Messages: messages,
		Stream:   true,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(b))
	}

	var sb strings.Builder
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
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			content := chunk.Choices[0].Delta.Content
			fmt.Print(content)
			sb.WriteString(content)
		}
	}
	fmt.Println()
	return sb.String(), scanner.Err()
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
