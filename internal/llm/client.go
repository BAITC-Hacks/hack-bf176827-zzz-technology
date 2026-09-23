// Package llm — тонкий клиент OpenAI Responses API (structured output + function calling) и файловый кэш.
// LLM не назначает роли: только формулирует текст по посчитанным фактам и отвечает на вопросы через инструменты.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Config struct {
	APIKey  string
	Model   string
	BaseURL string
	Timeout time.Duration
}

type Client struct {
	cfg  Config
	http *http.Client
}

func New(cfg Config) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.openai.com/v1"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}
	return &Client{cfg: cfg, http: &http.Client{Timeout: cfg.Timeout}}
}

func (c *Client) Enabled() bool { return c != nil && c.cfg.APIKey != "" }
func (c *Client) Model() string {
	if c == nil {
		return ""
	}
	return c.cfg.Model
}

// ToolExec выполняет инструмент и возвращает результат для модели (JSON или текст).
type ToolExec func(name, argsJSON string) (string, error)

// Complete — один запрос с ответом строго по JSON-схеме (schema=nil → свободный текст).
func (c *Client) Complete(ctx context.Context, system, user, schemaName string, schema map[string]any) (string, error) {
	if !c.Enabled() {
		return "", ErrDisabled
	}
	req := map[string]any{
		"model": c.cfg.Model,
		"input": []map[string]any{
			{"role": "developer", "content": system},
			{"role": "user", "content": user},
		},
	}
	if schema != nil {
		req["text"] = map[string]any{"format": map[string]any{
			"type": "json_schema", "name": schemaName, "schema": schema, "strict": true,
		}}
	}
	resp, err := c.call(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.text(), nil
}

// RunTools — цикл function calling: модель вызывает инструменты, результаты возвращаются через previous_response_id.
func (c *Client) RunTools(ctx context.Context, system, user string, tools []Tool, exec ToolExec, maxSteps int) (Answer, error) {
	if !c.Enabled() {
		return Answer{}, ErrDisabled
	}
	toolDefs := make([]map[string]any, len(tools))
	for i, t := range tools {
		toolDefs[i] = map[string]any{"type": "function", "name": t.Name, "description": t.Description, "parameters": t.Parameters}
	}
	req := map[string]any{
		"model": c.cfg.Model,
		"input": []map[string]any{
			{"role": "developer", "content": system},
			{"role": "user", "content": user},
		},
		"tools": toolDefs,
	}
	var ans Answer
	for step := 0; step < maxSteps; step++ {
		resp, err := c.call(ctx, req)
		if err != nil {
			return ans, err
		}
		ans.Steps++
		ans.Usage.InputTokens += resp.Usage.InputTokens
		ans.Usage.OutputTokens += resp.Usage.OutputTokens
		calls := resp.functionCalls()
		if len(calls) == 0 {
			ans.Text = resp.text()
			return ans, nil
		}
		var outputs []map[string]any
		for _, fc := range calls {
			out, err := exec(fc.Name, fc.Arguments)
			if err != nil {
				out = fmt.Sprintf(`{"error": %q}`, err.Error())
			}
			outputs = append(outputs, map[string]any{"type": "function_call_output", "call_id": fc.CallID, "output": out})
		}
		req = map[string]any{
			"model":                c.cfg.Model,
			"previous_response_id": resp.ID,
			"input":                outputs,
			"tools":                toolDefs,
		}
	}
	return ans, fmt.Errorf("llm: превышен лимит шагов (%d)", maxSteps)
}

func (r *response) text() string {
	var sb strings.Builder
	for _, o := range r.Output {
		if o.Type != "message" {
			continue
		}
		for _, c := range o.Content {
			if c.Type == "output_text" {
				sb.WriteString(c.Text)
			}
		}
	}
	return sb.String()
}

func (r *response) functionCalls() []functionCall {
	var out []functionCall
	for _, o := range r.Output {
		if o.Type == "function_call" {
			out = append(out, functionCall{Name: o.Name, CallID: o.CallID, Arguments: o.Arguments})
		}
	}
	return out
}

// call — POST /responses с одним повтором на сетевую ошибку или 5xx/429.
func (c *Client) call(ctx context.Context, body map[string]any) (*response, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		if attempt > 0 {
			time.Sleep(2 * time.Second)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/responses", bytes.NewReader(b))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
		req.Header.Set("Content-Type", "application/json")
		httpResp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		data, _ := io.ReadAll(httpResp.Body)
		httpResp.Body.Close()
		if httpResp.StatusCode == 429 || httpResp.StatusCode >= 500 {
			lastErr = fmt.Errorf("llm: http %d: %s", httpResp.StatusCode, truncate(string(data), 300))
			continue
		}
		var r response
		if err := json.Unmarshal(data, &r); err != nil {
			return nil, fmt.Errorf("llm: bad json: %w", err)
		}
		if httpResp.StatusCode >= 400 || r.Error != nil {
			msg := truncate(string(data), 300)
			if r.Error != nil {
				msg = r.Error.Message
			}
			return nil, fmt.Errorf("llm: http %d: %s", httpResp.StatusCode, msg)
		}
		return &r, nil
	}
	return nil, lastErr
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
