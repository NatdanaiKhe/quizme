package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	// requestTimeout bounds a single AI call.
	requestTimeout = 60 * time.Second
	// maxTokens caps worst-case response length/cost.
	// 5 questions × (~250 tokens prompt+options+explanation+JSON overhead) ≈ 1250; 2000 gives headroom.
	maxTokens = 2000
)

// AIError means the request never produced usable content (transport, non-200, empty/malformed envelope).
type AIError struct {
	StatusCode int
	Message    string
}

func (e *AIError) Error() string {
	if e.StatusCode != 0 {
		return fmt.Sprintf("ai request failed (status %d): %s", e.StatusCode, e.Message)
	}
	return "ai request failed: " + e.Message
}

// IsAIError reports whether err is an *AIError.
func IsAIError(err error) bool {
	var a *AIError
	return errors.As(err, &a)
}

// Client calls an OpenAI-compatible chat completions endpoint.
type Client struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

// New creates a Client. baseURL should be the API root (e.g. https://api.openai.com/v1).
func New(baseURL, apiKey, model string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{Timeout: requestTimeout},
	}
}

// Generated calls the AI, parses the response, and validates it.
// It returns validated questions and total tokens used by the API.
func (c *Client) Generate(ctx context.Context, n int, topics []string) ([]GeneratedQuestion, int, error) {
	content, tokens, err := c.generateRaw(ctx, n, topics)
	if err != nil {
		return nil, 0, err
	}
	qs, err := Parse(content, n)
	if err != nil {
		return nil, tokens, err
	}
	if err := Validate(qs, n, topics); err != nil {
		return nil, tokens, err
	}
	return qs, tokens, nil
}

type chatRequest struct {
	Model     string    `json:"model"`
	Messages  []message `json:"messages"`
	MaxTokens int       `json:"max_tokens"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

func (c *Client) generateRaw(ctx context.Context, n int, topics []string) (string, int, error) {
	if c.baseURL == "" {
		return "", 0, &AIError{Message: "AI base URL not configured"}
	}
	if c.apiKey == "" {
		return "", 0, &AIError{Message: "AI API key not configured"}
	}
	if c.model == "" {
		return "", 0, &AIError{Message: "AI model not configured"}
	}

	reqBody := chatRequest{
		Model: c.model,
		Messages: []message{
			{Role: "system", Content: buildPrompt(n, topics)},
		},
		MaxTokens: maxTokens,
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, &AIError{Message: "marshal request: " + err.Error()}
	}

	url := c.baseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", 0, &AIError{Message: "create request: " + err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", 0, &AIError{Message: "transport: " + err.Error()}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, &AIError{Message: "read body: " + err.Error()}
	}

	if resp.StatusCode != http.StatusOK {
		return "", 0, &AIError{
			StatusCode: resp.StatusCode,
			Message:    truncate(string(body), 200),
		}
	}

	var env chatResponse
	if err := json.Unmarshal(body, &env); err != nil {
		return "", 0, &AIError{Message: "unmarshal envelope: " + err.Error() + " body: " + truncate(string(body), 200)}
	}
	if len(env.Choices) == 0 || strings.TrimSpace(env.Choices[0].Message.Content) == "" {
		return "", 0, &AIError{Message: "empty choices/content in response"}
	}
	return env.Choices[0].Message.Content, env.Usage.TotalTokens, nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
