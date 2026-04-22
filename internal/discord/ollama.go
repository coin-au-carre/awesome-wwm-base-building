package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
)

// OllamaResponder calls a local Ollama instance and maintains per-channel conversation history.
type OllamaResponder struct {
	baseURL      string // e.g. http://localhost:11434
	model        string // e.g. llama3.2-uncensored
	mu           sync.Mutex
	history      map[string][]ollamaMessage // API mode
	systemPrompt string
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	System   string          `json:"system"`
}

type ollamaChatResponse struct {
	Message ollamaMessage `json:"message"`
}

// NewOllamaResponder creates a responder that uses a local Ollama instance.
// baseURL should be like http://localhost:11434
// model should be like llama3.2-uncensored or llama3.1
func NewOllamaResponder(baseURL, model, root string) *OllamaResponder {
	return &OllamaResponder{
		baseURL:      strings.TrimSuffix(baseURL, "/"),
		model:        model,
		history:      make(map[string][]ollamaMessage),
		systemPrompt: buildSystemPrompt(root),
	}
}

// Reply calls Ollama and maintains conversation history per channel.
func (r *OllamaResponder) Reply(ctx context.Context, channelID, userMessage string, imageURLs []string) (Result, error) {
	// Note: Ollama doesn't support image URLs in the standard chat API, so we log and ignore them.
	if len(imageURLs) > 0 {
		slog.Info("ollama: image URLs ignored (not supported)", "count", len(imageURLs))
	}

	r.mu.Lock()
	msgs := make([]ollamaMessage, len(r.history[channelID]))
	copy(msgs, r.history[channelID])
	r.mu.Unlock()

	msgs = append(msgs, ollamaMessage{Role: "user", Content: userMessage})

	reqBody := ollamaChatRequest{
		Model:    r.model,
		Messages: msgs,
		Stream:   false,
		System:   r.systemPrompt,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return Result{}, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", r.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return Result{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("ollama request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return Result{}, fmt.Errorf("ollama API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var respData ollamaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return Result{}, fmt.Errorf("decode response: %w", err)
	}

	text := respData.Message.Content

	// Update history with assistant response.
	msgs = append(msgs, ollamaMessage{Role: "assistant", Content: text})

	// Trim history to maxHistory messages.
	if len(msgs) > maxHistory {
		msgs = msgs[len(msgs)-maxHistory:]
	}

	r.mu.Lock()
	r.history[channelID] = msgs
	r.mu.Unlock()

	// Parse text sentinels (since Ollama doesn't support tool use).
	return parseCLIResult(text), nil
}

// Caption generates a short in-character reaction to a guild spotlight using Ollama.
func (r *OllamaResponder) Caption(ctx context.Context, guildName string, tags []string) string {
	prompt := fmt.Sprintf("React in one tiny sentence to this guild base spotlight: %s", guildName)
	if len(tags) > 0 {
		prompt += fmt.Sprintf(" (tags: %s)", strings.Join(tags, ", "))
	}

	reqBody := ollamaChatRequest{
		Model:   r.model,
		Messages: []ollamaMessage{{Role: "user", Content: prompt}},
		Stream:  false,
		System:  r.systemPrompt,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return ""
	}

	req, err := http.NewRequestWithContext(ctx, "POST", r.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return ""
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var respData ollamaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return ""
	}

	return removeBlankLines(respData.Message.Content)
}
