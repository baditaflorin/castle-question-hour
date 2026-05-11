// Package summarize takes a hour's worth of anonymous answers and asks a
// local LLM (Ollama) to extract 3-7 themes.
package summarize

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

var ErrLLMTimeout = errors.New("llm timed out")

// Theme is one extracted theme with optional sample phrases.
type Theme struct {
	Title   string   `json:"title"`
	Summary string   `json:"summary"`
	Samples []string `json:"samples,omitempty"`
}

// Result is the structured output the frontend renders.
type Result struct {
	Themes      []Theme `json:"themes"`
	AnswerCount int     `json:"answer_count"`
}

// Client talks HTTP to an Ollama server.
type Client struct {
	URL     string
	Model   string
	Timeout time.Duration
	HTTP    *http.Client
}

func New(url, model string, timeout time.Duration) *Client {
	return &Client{
		URL:     strings.TrimRight(url, "/"),
		Model:   model,
		Timeout: timeout,
		HTTP:    &http.Client{Timeout: timeout + 5*time.Second},
	}
}

const systemPrompt = `You are a careful, warm note-taker at a castle gathering. ` +
	`Given anonymous one-paragraph reflections from many guests, extract 3 to 7 themes ` +
	`that genuinely repeat. Do not invent themes. Be specific, not generic. ` +
	`Return ONLY JSON of shape: {"themes":[{"title":"string","summary":"string","samples":["string"]}]}.`

// Summarize calls Ollama /api/generate with a JSON-mode prompt and parses the result.
func (c *Client) Summarize(ctx context.Context, question string, answers []string) (*Result, error) {
	if c.URL == "" {
		return nil, errors.New("OLLAMA_URL not configured")
	}
	if len(answers) == 0 {
		return &Result{Themes: nil, AnswerCount: 0}, nil
	}

	body := map[string]any{
		"model":  c.Model,
		"system": systemPrompt,
		"prompt": buildPrompt(question, answers),
		"format": "json",
		"stream": false,
		"options": map[string]any{
			"temperature": 0.2,
			"num_predict": 1024,
		},
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal ollama request: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL+"/api/generate", bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, ErrLLMTimeout
		}
		return nil, fmt.Errorf("ollama request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // drained via Close
	if resp.StatusCode/100 != 2 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("ollama %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var wrapper struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, fmt.Errorf("decode ollama wrapper: %w", err)
	}
	var inner Result
	if err := json.Unmarshal([]byte(wrapper.Response), &inner); err != nil {
		return nil, fmt.Errorf("decode themes payload: %w", err)
	}
	inner.AnswerCount = len(answers)
	clamp(&inner)
	return &inner, nil
}

func buildPrompt(question string, answers []string) string {
	var b strings.Builder
	b.WriteString("Question of the hour: ")
	b.WriteString(question)
	b.WriteString("\n\nAnonymous answers:\n")
	for i, a := range answers {
		fmt.Fprintf(&b, "%d. %s\n", i+1, strings.TrimSpace(a))
	}
	b.WriteString("\nReturn the JSON themes object only.")
	return b.String()
}

// clamp enforces the 3-7 themes constraint defensively, in case the model overshoots.
func clamp(r *Result) {
	if len(r.Themes) > 7 {
		r.Themes = r.Themes[:7]
	}
}
