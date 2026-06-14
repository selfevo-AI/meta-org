package aigateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type AnthropicAdapter struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

const defaultAnthropicBaseURL = "https://api.anthropic.com"

func NewAnthropicAdapter(baseURL string, apiKey string, client *http.Client) *AnthropicAdapter {
	if baseURL == "" {
		baseURL = defaultAnthropicBaseURL
	}
	return &AnthropicAdapter{
		baseURL: baseURL,
		apiKey:  apiKey,
		client:  httpClientOrDefault(client),
	}
}

func (a *AnthropicAdapter) Invoke(ctx context.Context, req ProviderRequest) (*ProviderResponse, error) {
	payload := a.payload(req, false)
	resp, err := postProviderJSON(ctx, a.client, joinEndpoint(a.baseURL, "/v1/messages"), a.headers(), payload)
	if err != nil {
		return nil, err
	}

	var decoded anthropicMessageResponse
	if err := decodeProviderJSON("anthropic", resp, &decoded); err != nil {
		return nil, err
	}
	return decoded.toProviderResponse(), nil
}

func (a *AnthropicAdapter) Stream(ctx context.Context, req ProviderRequest) (<-chan StreamEvent, error) {
	payload := a.payload(req, true)
	resp, err := postProviderJSON(ctx, a.client, joinEndpoint(a.baseURL, "/v1/messages"), a.headers(), payload)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		defer resp.Body.Close()
		return nil, providerHTTPError("anthropic", resp)
	}
	return streamProviderEvents(ctx, resp.Body, anthropicStreamEvents), nil
}

func (a *AnthropicAdapter) headers() map[string]string {
	return map[string]string{
		"x-api-key":         a.apiKey,
		"anthropic-version": "2023-06-01",
	}
}

func (a *AnthropicAdapter) payload(req ProviderRequest, stream bool) map[string]any {
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 1024
	}
	payload := map[string]any{
		"model":      req.Model,
		"messages":   anthropicMessages(req.Messages),
		"max_tokens": maxTokens,
		"stream":     stream,
	}
	if req.Temperature != nil {
		payload["temperature"] = *req.Temperature
	}
	if len(req.Tools) > 0 {
		payload["tools"] = anthropicTools(req.Tools)
	}
	return payload
}

func anthropicMessages(messages []Message) []map[string]any {
	result := make([]map[string]any, 0, len(messages))
	for _, msg := range messages {
		if msg.Role == "system" {
			continue
		}
		role := msg.Role
		if role == "assistant" {
			role = "assistant"
		}
		result = append(result, map[string]any{
			"role":    role,
			"content": msg.Content,
		})
	}
	return result
}

func anthropicTools(tools []ToolDefinition) []map[string]any {
	result := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		result = append(result, map[string]any{
			"name":         tool.Name,
			"description":  tool.Description,
			"input_schema": copyMap(tool.Schema),
		})
	}
	return result
}

type anthropicMessageResponse struct {
	ID      string                  `json:"id"`
	Content []anthropicContentBlock `json:"content"`
	Usage   struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type anthropicContentBlock struct {
	Type  string         `json:"type"`
	Text  string         `json:"text"`
	ID    string         `json:"id"`
	Name  string         `json:"name"`
	Input map[string]any `json:"input"`
}

func (r anthropicMessageResponse) toProviderResponse() *ProviderResponse {
	resp := &ProviderResponse{
		ProviderRequestID: r.ID,
		Usage: TokenUsage{
			InputTokens:  r.Usage.InputTokens,
			OutputTokens: r.Usage.OutputTokens,
		},
	}
	for _, block := range r.Content {
		switch block.Type {
		case "text":
			resp.Content += block.Text
		case "tool_use":
			resp.ToolCalls = append(resp.ToolCalls, ToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: copyMap(block.Input),
			})
		}
	}
	return resp
}

func anthropicStreamEvents(raw RawSSEEvent) []StreamEvent {
	var event anthropicStreamEvent
	if err := json.Unmarshal([]byte(raw.Data), &event); err != nil {
		return []StreamEvent{{Type: "error", Error: fmt.Sprintf("decode anthropic stream: %v", err)}}
	}
	eventType := event.Type
	if eventType == "" {
		eventType = raw.Event
	}
	switch eventType {
	case "content_block_delta":
		if event.Delta.Text != "" {
			return []StreamEvent{{Type: "delta", Delta: event.Delta.Text}}
		}
	case "message_delta":
		if event.Usage.OutputTokens > 0 {
			return []StreamEvent{{Type: "usage", Usage: TokenUsage{OutputTokens: event.Usage.OutputTokens}}}
		}
	case "message_stop":
		return []StreamEvent{{Type: "done", Done: true}}
	case "error":
		if event.Error.Message != "" {
			return []StreamEvent{{Type: "error", Error: event.Error.Message}}
		}
	}
	return nil
}

type anthropicStreamEvent struct {
	Type  string `json:"type"`
	Delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta"`
	Usage struct {
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}
