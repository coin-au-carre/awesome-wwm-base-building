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

// ollamaNumCtx is the context window size passed to Ollama.
// The system prompt alone is ~12K tokens (guild directory + catalog), so we
// need at least 16K. 32K fits comfortably on 8 GB RAM with a 7B model.
const ollamaNumCtx = 32768

// OllamaResponder calls a local Ollama instance and maintains per-channel conversation history.
type OllamaResponder struct {
	baseURL      string
	model        string
	mu           sync.Mutex
	history      map[string][]ollamaMessage // user/assistant pairs only; system is prepended per-call
	systemPrompt string
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatRequest struct {
	Model    string         `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool           `json:"stream"`
	Options  map[string]any `json:"options,omitempty"`
}

type ollamaChatResponse struct {
	Message ollamaMessage `json:"message"`
}

// NewOllamaResponder creates a responder that uses a local Ollama instance.
func NewOllamaResponder(baseURL, model, root string) *OllamaResponder {
	prompt := buildSystemPrompt(root)
	slog.Info("ollama: system prompt loaded", "chars", len(prompt), "approx_tokens", len(prompt)/4)
	return &OllamaResponder{
		baseURL:      strings.TrimSuffix(baseURL, "/"),
		model:        model,
		history:      make(map[string][]ollamaMessage),
		systemPrompt: prompt,
	}
}

// Reply calls Ollama and maintains conversation history per channel.
func (r *OllamaResponder) Reply(ctx context.Context, channelID, userMessage string, imageURLs []string) (Result, error) {
	if len(imageURLs) > 0 {
		slog.Info("ollama: image URLs ignored (not supported)", "count", len(imageURLs))
	}

	r.mu.Lock()
	history := make([]ollamaMessage, len(r.history[channelID]))
	copy(history, r.history[channelID])
	r.mu.Unlock()

	// System message is prepended fresh every call — more reliable than the top-level system field.
	msgs := make([]ollamaMessage, 0, 1+len(history)+1)
	msgs = append(msgs, ollamaMessage{Role: "system", Content: r.systemPrompt})
	msgs = append(msgs, history...)
	msgs = append(msgs, ollamaMessage{Role: "user", Content: userMessage})

	reqBody := ollamaChatRequest{
		Model:   r.model,
		Messages: msgs,
		Stream:  false,
		Options: map[string]any{"num_ctx": ollamaNumCtx},
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

	// Persist only user/assistant history (system is always prepended fresh).
	newHistory := append(history, ollamaMessage{Role: "user", Content: userMessage}, ollamaMessage{Role: "assistant", Content: text})
	if len(newHistory) > maxHistory {
		newHistory = newHistory[len(newHistory)-maxHistory:]
	}

	r.mu.Lock()
	r.history[channelID] = newHistory
	r.mu.Unlock()

	return parseCLIResult(text), nil
}

// Caption generates a short in-character reaction to a guild spotlight using Ollama.
func (r *OllamaResponder) Caption(ctx context.Context, guildName string, tags []string) string {
	prompt := fmt.Sprintf("React in one tiny sentence to this guild base spotlight: %s", guildName)
	if len(tags) > 0 {
		prompt += fmt.Sprintf(" (tags: %s)", strings.Join(tags, ", "))
	}

	reqBody := ollamaChatRequest{
		Model: r.model,
		Messages: []ollamaMessage{
			{Role: "system", Content: r.systemPrompt},
			{Role: "user", Content: prompt},
		},
		Stream:  false,
		Options: map[string]any{"num_ctx": ollamaNumCtx},
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
