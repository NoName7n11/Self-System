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

type OpenAIProvider struct {
	settings ProviderSettings
	client   *http.Client
}

func NewOpenAIProvider(settings ProviderSettings) *OpenAIProvider {
	if settings.Model == "" {
		settings.Model = "gpt-4o-mini"
	}
	if settings.BaseURL == "" {
		settings.BaseURL = "https://api.openai.com"
	}
	if settings.TimeoutSeconds <= 0 {
		settings.TimeoutSeconds = 20
	}

	return &OpenAIProvider{
		settings: settings,
		client: &http.Client{
			Timeout: time.Duration(settings.TimeoutSeconds) * time.Second,
		},
	}
}

func (p *OpenAIProvider) Name() string {
	return "openai"
}

func (p *OpenAIProvider) ClassifySkim(ctx context.Context, input ClassificationInput) (ClassificationOutput, error) {
	if !p.settings.Enabled || strings.TrimSpace(p.settings.APIKey) == "" {
		return ClassificationOutput{}, ErrProviderUnavailable
	}

	endpoint := strings.TrimRight(p.settings.BaseURL, "/") + "/v1/chat/completions"
	requestBody := map[string]any{
		"model": p.settings.Model,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "You are a strict JSON classification service. Output valid JSON only.",
			},
			{
				"role":    "user",
				"content": buildClassificationPrompt(input),
			},
		},
		"temperature": 0.1,
	}

	payload, err := json.Marshal(requestBody)
	if err != nil {
		return ClassificationOutput{}, fmt.Errorf("marshal openai request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return ClassificationOutput{}, fmt.Errorf("build openai request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.settings.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return ClassificationOutput{}, fmt.Errorf("openai request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ClassificationOutput{}, fmt.Errorf("read openai response: %w", err)
	}
	if resp.StatusCode >= 300 {
		return ClassificationOutput{}, fmt.Errorf("openai status %d: %s", resp.StatusCode, string(body))
	}

	var decoded struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return ClassificationOutput{}, fmt.Errorf("decode openai response: %w", err)
	}
	if len(decoded.Choices) == 0 {
		return ClassificationOutput{}, fmt.Errorf("openai returned no choices")
	}

	return parseModelOutput(decoded.Choices[0].Message.Content)
}
