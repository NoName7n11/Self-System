package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type GeminiProvider struct {
	settings ProviderSettings
	client   *http.Client
}

func NewGeminiProvider(settings ProviderSettings) *GeminiProvider {
	if settings.Model == "" {
		settings.Model = "gemini-1.5-flash"
	}
	if settings.BaseURL == "" {
		settings.BaseURL = "https://generativelanguage.googleapis.com"
	}
	if settings.TimeoutSeconds <= 0 {
		settings.TimeoutSeconds = 20
	}

	return &GeminiProvider{
		settings: settings,
		client: &http.Client{
			Timeout: time.Duration(settings.TimeoutSeconds) * time.Second,
		},
	}
}

func (p *GeminiProvider) Name() string {
	return "gemini"
}

func (p *GeminiProvider) ClassifySkim(ctx context.Context, input ClassificationInput) (ClassificationOutput, error) {
	if !p.settings.Enabled || strings.TrimSpace(p.settings.APIKey) == "" {
		return ClassificationOutput{}, ErrProviderUnavailable
	}

	endpoint := strings.TrimRight(p.settings.BaseURL, "/") +
		"/v1beta/models/" + url.PathEscape(p.settings.Model) +
		":generateContent"

	prompt := "You are a strict JSON classification service. Output valid JSON only.\n\n" + buildClassificationPrompt(input)
	requestBody := map[string]any{
		"contents": []map[string]any{
			{
				"parts": []map[string]string{{"text": prompt}},
			},
		},
		"generationConfig": map[string]any{
			"temperature": 0.1,
		},
	}

	payload, err := json.Marshal(requestBody)
	if err != nil {
		return ClassificationOutput{}, fmt.Errorf("marshal gemini request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return ClassificationOutput{}, fmt.Errorf("build gemini request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", strings.TrimSpace(p.settings.APIKey))

	resp, err := p.client.Do(req)
	if err != nil {
		return ClassificationOutput{}, fmt.Errorf("gemini request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ClassificationOutput{}, fmt.Errorf("read gemini response: %w", err)
	}
	if resp.StatusCode >= 300 {
		return ClassificationOutput{}, fmt.Errorf("gemini status %d: %s", resp.StatusCode, string(body))
	}

	var decoded struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return ClassificationOutput{}, fmt.Errorf("decode gemini response: %w", err)
	}
	if len(decoded.Candidates) == 0 || len(decoded.Candidates[0].Content.Parts) == 0 {
		return ClassificationOutput{}, fmt.Errorf("gemini returned no content")
	}

	return parseModelOutput(decoded.Candidates[0].Content.Parts[0].Text)
}
