package aigateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type OpenAIAdapter struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

const defaultOpenAIBaseURL = "https://api.openai.com"

func NewOpenAIAdapter(baseURL string, apiKey string, client *http.Client) *OpenAIAdapter {
	if baseURL == "" {
		baseURL = defaultOpenAIBaseURL
	}
	return &OpenAIAdapter{
		baseURL: baseURL,
		apiKey:  apiKey,
		client:  httpClientOrDefault(client),
	}
}

func (a *OpenAIAdapter) Invoke(ctx context.Context, req ProviderRequest) (*ProviderResponse, error) {
	payload := a.payload(req, false)
	resp, err := postProviderJSON(ctx, a.client, joinEndpoint(a.baseURL, "/v1/chat/completions"), a.headers(), payload)
	if err != nil {
		return nil, err
	}

	var decoded openAIChatResponse
	if err := decodeProviderJSON("openai", resp, &decoded); err != nil {
		return nil, err
	}
	return decoded.toProviderResponse(), nil
}

func (a *OpenAIAdapter) Stream(ctx context.Context, req ProviderRequest) (<-chan StreamEvent, error) {
	payload := a.payload(req, true)
	resp, err := postProviderJSON(ctx, a.client, joinEndpoint(a.baseURL, "/v1/chat/completions"), a.headers(), payload)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		defer resp.Body.Close()
		return nil, providerHTTPError("openai", resp)
	}
	return streamProviderEvents(ctx, resp.Body, openAIStreamEvents), nil
}

func (a *OpenAIAdapter) headers() map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + a.apiKey,
	}
}

func (a *OpenAIAdapter) payload(req ProviderRequest, stream bool) map[string]any {
	payload := map[string]any{
		"model":    req.Model,
		"messages": openAIMessages(req.Messages),
		"stream":   stream,
	}
	if req.Temperature != nil {
		payload["temperature"] = *req.Temperature
	}
	if req.MaxTokens > 0 {
		payload["max_tokens"] = req.MaxTokens
	}
	if len(req.Tools) > 0 {
		payload["tools"] = openAITools(req.Tools)
	}
	return payload
}

func openAIMessages(messages []Message) []map[string]any {
	result := make([]map[string]any, 0, len(messages))
	for _, msg := range messages {
		item := map[string]any{"role": msg.Role}
		switch msg.Role {
		case "tool":
			item["content"] = msg.Content
			item["tool_call_id"] = msg.ToolCallID
		default:
			item["content"] = msg.Content
			if len(msg.ToolCalls) > 0 {
				calls := make([]map[string]any, 0, len(msg.ToolCalls))
				for i, call := range msg.ToolCalls {
					args, _ := json.Marshal(call.Arguments)
					calls = append(calls, map[string]any{
						"id":   call.normalizedID(i),
						"type": "function",
						"function": map[string]any{
							"name":      call.Name,
							"arguments": string(args),
						},
					})
				}
				item["tool_calls"] = calls
			}
		}
		result = append(result, item)
	}
	return result
}

func openAITools(tools []ToolDefinition) []map[string]any {
	result := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		result = append(result, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        tool.Name,
				"description": tool.Description,
				"parameters":  copyMap(tool.Schema),
			},
		})
	}
	return result
}

type openAIChatResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message struct {
			Content   string           `json:"content"`
			ToolCalls []openAIToolCall `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
	Usage openAIUsage `json:"usage"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

type openAIToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

func (r openAIChatResponse) toProviderResponse() *ProviderResponse {
	resp := &ProviderResponse{
		ProviderRequestID: r.ID,
		Usage: TokenUsage{
			InputTokens:  r.Usage.PromptTokens,
			OutputTokens: r.Usage.CompletionTokens,
		},
	}
	for _, choice := range r.Choices {
		resp.Content += choice.Message.Content
		for _, toolCall := range choice.Message.ToolCalls {
			resp.ToolCalls = append(resp.ToolCalls, toolCall.toToolCall())
		}
	}
	return resp
}

func (c openAIToolCall) toToolCall() ToolCall {
	return ToolCall{
		ID:        c.ID,
		Name:      c.Function.Name,
		Arguments: parseToolArguments(c.Function.Arguments),
	}
}

func openAIStreamEvents(raw RawSSEEvent) []StreamEvent {
	if raw.Done {
		return []StreamEvent{{Type: "done", Done: true}}
	}
	var chunk openAIStreamChunk
	if err := json.Unmarshal([]byte(raw.Data), &chunk); err != nil {
		return []StreamEvent{{Type: "error", Error: fmt.Sprintf("decode openai stream: %v", err)}}
	}
	events := []StreamEvent{}
	for _, choice := range chunk.Choices {
		if choice.Delta.Content != "" {
			events = append(events, StreamEvent{Type: "delta", Delta: choice.Delta.Content})
		}
		for _, toolCall := range choice.Delta.ToolCalls {
			call := toolCall.toToolCall()
			events = append(events, StreamEvent{Type: "tool_call", ToolCall: &call})
		}
		if choice.FinishReason == "stop" {
			events = append(events, StreamEvent{Type: "done", Done: true})
		}
	}
	if chunk.Usage.PromptTokens > 0 || chunk.Usage.CompletionTokens > 0 {
		events = append(events, StreamEvent{Type: "usage", Usage: TokenUsage{InputTokens: chunk.Usage.PromptTokens, OutputTokens: chunk.Usage.CompletionTokens}})
	}
	return events
}

type openAIStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content   string           `json:"content"`
			ToolCalls []openAIToolCall `json:"tool_calls"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage openAIUsage `json:"usage"`
}
