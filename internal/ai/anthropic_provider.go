package ai

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

type AnthropicProvider struct {
	settings ProviderSettings
	client   *http.Client
}

func NewAnthropicProvider(settings ProviderSettings) *AnthropicProvider {
	if settings.Model == "" {
		settings.Model = "claude-3-5-sonnet-latest"
	}
	if settings.BaseURL == "" {
		settings.BaseURL = "https://api.anthropic.com"
	}
	if settings.TimeoutSeconds <= 0 {
		settings.TimeoutSeconds = 20
	}

	return &AnthropicProvider{
		settings: settings,
		client: &http.Client{
			Timeout: time.Duration(settings.TimeoutSeconds) * time.Second,
		},
	}
}

func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

func (p *AnthropicProvider) ClassifySkim(ctx context.Context, input ClassificationInput) (ClassificationOutput, error) {
	if !p.settings.Enabled || strings.TrimSpace(p.settings.APIKey) == "" {
		return ClassificationOutput{}, ErrProviderUnavailable
	}

	endpoint := strings.TrimRight(p.settings.BaseURL, "/") + "/v1/messages"
	requestBody := map[string]any{
		"model":       p.settings.Model,
		"max_tokens":  300,
		"temperature": 0.1,
		"system":      "You are a strict JSON classification service. Output valid JSON only.",
		"messages": []map[string]any{
			{
				"role":    "user",
				"content": buildClassificationPrompt(input),
			},
		},
	}

	payload, err := json.Marshal(requestBody)
	if err != nil {
		return ClassificationOutput{}, fmt.Errorf("marshal anthropic request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return ClassificationOutput{}, fmt.Errorf("build anthropic request: %w", err)
	}
	req.Header.Set("x-api-key", p.settings.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return ClassificationOutput{}, fmt.Errorf("anthropic request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ClassificationOutput{}, fmt.Errorf("read anthropic response: %w", err)
	}
	if resp.StatusCode >= 300 {
		return ClassificationOutput{}, fmt.Errorf("anthropic status %d: %s", resp.StatusCode, string(body))
	}

	var decoded struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return ClassificationOutput{}, fmt.Errorf("decode anthropic response: %w", err)
	}
	if len(decoded.Content) == 0 {
		return ClassificationOutput{}, fmt.Errorf("anthropic returned no content")
	}

	return parseModelOutput(decoded.Content[0].Text)
}
