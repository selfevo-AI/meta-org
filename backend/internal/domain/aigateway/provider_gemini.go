package aigateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type GeminiAdapter struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

const defaultGeminiBaseURL = "https://generativelanguage.googleapis.com"

func NewGeminiAdapter(baseURL string, apiKey string, client *http.Client) *GeminiAdapter {
	if baseURL == "" {
		baseURL = defaultGeminiBaseURL
	}
	return &GeminiAdapter{
		baseURL: baseURL,
		apiKey:  apiKey,
		client:  httpClientOrDefault(client),
	}
}

func (a *GeminiAdapter) Invoke(ctx context.Context, req ProviderRequest) (*ProviderResponse, error) {
	resp, err := postProviderJSON(ctx, a.client, a.endpoint(req.Model, false), nil, a.payload(req))
	if err != nil {
		return nil, err
	}

	var decoded geminiResponse
	if err := decodeProviderJSON("gemini", resp, &decoded); err != nil {
		return nil, err
	}
	providerResp := decoded.toProviderResponse()
	providerResp.ProviderRequestID = resp.Header.Get("x-request-id")
	return providerResp, nil
}

func (a *GeminiAdapter) Stream(ctx context.Context, req ProviderRequest) (<-chan StreamEvent, error) {
	resp, err := postProviderJSON(ctx, a.client, a.endpoint(req.Model, true), nil, a.payload(req))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		defer resp.Body.Close()
		return nil, providerHTTPError("gemini", resp)
	}
	return streamProviderEvents(ctx, resp.Body, geminiStreamEvents), nil
}

func (a *GeminiAdapter) endpoint(model string, stream bool) string {
	action := "generateContent"
	if stream {
		action = "streamGenerateContent"
	}
	path := fmt.Sprintf("/v1beta/models/%s:%s", url.PathEscape(model), action)
	values := url.Values{"key": []string{a.apiKey}}
	if stream {
		values.Set("alt", "sse")
	}
	return joinEndpoint(a.baseURL, path) + "?" + values.Encode()
}

func (a *GeminiAdapter) payload(req ProviderRequest) map[string]any {
	payload := map[string]any{
		"contents": geminiContents(req.Messages),
	}
	if req.Temperature != nil || req.MaxTokens > 0 {
		config := map[string]any{}
		if req.Temperature != nil {
			config["temperature"] = *req.Temperature
		}
		if req.MaxTokens > 0 {
			config["maxOutputTokens"] = req.MaxTokens
		}
		payload["generationConfig"] = config
	}
	if len(req.Tools) > 0 {
		payload["tools"] = geminiTools(req.Tools)
	}
	return payload
}

func geminiContents(messages []Message) []map[string]any {
	result := make([]map[string]any, 0, len(messages))
	for _, msg := range messages {
		role := "user"
		if msg.Role == "assistant" || msg.Role == "model" {
			role = "model"
		}
		parts := []map[string]any{}
		if msg.Role == "tool" {
			name := msg.ToolName
			if name == "" {
				name = msg.ToolCallID
			}
			parts = append(parts, map[string]any{
				"functionResponse": map[string]any{
					"name":     name,
					"response": map[string]any{"result": msg.Content},
				},
			})
		} else {
			if msg.Content != "" {
				parts = append(parts, map[string]any{"text": msg.Content})
			}
			for _, call := range msg.ToolCalls {
				parts = append(parts, map[string]any{
					"functionCall": map[string]any{
						"name": call.Name,
						"args": copyMap(call.Arguments),
					},
				})
			}
		}
		if len(parts) == 0 {
			parts = append(parts, map[string]any{"text": ""})
		}
		result = append(result, map[string]any{
			"role":  role,
			"parts": parts,
		})
	}
	return result
}

func geminiTools(tools []ToolDefinition) []map[string]any {
	declarations := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		declarations = append(declarations, map[string]any{
			"name":        tool.Name,
			"description": tool.Description,
			"parameters":  copyMap(tool.Schema),
		})
	}
	return []map[string]any{{"functionDeclarations": declarations}}
}

type geminiResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text         string `json:"text"`
	FunctionCall *struct {
		Name string         `json:"name"`
		Args map[string]any `json:"args"`
	} `json:"functionCall"`
}

func (r geminiResponse) toProviderResponse() *ProviderResponse {
	resp := &ProviderResponse{
		Usage: TokenUsage{
			InputTokens:  r.UsageMetadata.PromptTokenCount,
			OutputTokens: r.UsageMetadata.CandidatesTokenCount,
		},
	}
	for _, candidate := range r.Candidates {
		for _, part := range candidate.Content.Parts {
			resp.Content += part.Text
			if part.FunctionCall != nil {
				resp.ToolCalls = append(resp.ToolCalls, ToolCall{
					Name:      part.FunctionCall.Name,
					Arguments: copyMap(part.FunctionCall.Args),
				})
			}
		}
	}
	return resp
}

func geminiStreamEvents(raw RawSSEEvent) []StreamEvent {
	var chunk geminiResponse
	if err := json.Unmarshal([]byte(raw.Data), &chunk); err != nil {
		return []StreamEvent{{Type: "error", Error: fmt.Sprintf("decode gemini stream: %v", err)}}
	}
	events := []StreamEvent{}
	resp := chunk.toProviderResponse()
	if resp.Content != "" {
		events = append(events, StreamEvent{Type: "delta", Delta: resp.Content})
	}
	for _, toolCall := range resp.ToolCalls {
		call := toolCall
		events = append(events, StreamEvent{Type: "tool_call", ToolCall: &call})
	}
	if resp.Usage.InputTokens > 0 || resp.Usage.OutputTokens > 0 {
		events = append(events, StreamEvent{Type: "usage", Usage: resp.Usage})
	}
	return events
}
