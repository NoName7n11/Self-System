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

// EnrichmentInput contains the content to be enriched by the AI.
type EnrichmentInput struct {
	Title   string
	URL     string
	Content string // extracted text from Change 6 (main_text / pdf_text / ocr_text)
}

// EnrichmentResult is the structured output of an AI enrichment call.
type EnrichmentResult struct {
	Summary   string
	KeyPoints []string
	Entities  []string
	// Provider that produced this result.
	Provider string
}

// EnrichmentProvider produces AI-generated summaries, key points, and entities.
type EnrichmentProvider interface {
	Name() string
	Enrich(ctx context.Context, input EnrichmentInput) (EnrichmentResult, error)
}

// RegisterEnrichment adds an enrichment provider to the manager. Providers are
// tried in registration order; the first to succeed wins.
func (m *Manager) RegisterEnrichment(provider EnrichmentProvider) {
	if provider == nil {
		return
	}
	m.enrichmentProviders = append(m.enrichmentProviders, provider)
}

// EnrichResource produces a summary, key points, and entities for the given
// content by trying each registered enrichment provider in order. Falls back
// to ErrProviderUnavailable if no provider can handle the request.
func (m *Manager) EnrichResource(ctx context.Context, input EnrichmentInput) (EnrichmentResult, error) {
	var lastErr error
	for _, p := range m.enrichmentProviders {
		result, err := p.Enrich(ctx, input)
		if err != nil {
			if err == ErrProviderUnavailable {
				continue
			}
			lastErr = err
			continue
		}
		result.Provider = p.Name()
		return result, nil
	}
	if lastErr != nil {
		return EnrichmentResult{}, lastErr
	}
	return EnrichmentResult{}, ErrProviderUnavailable
}

// ---- OpenAI enrichment provider --------------------------------------------

// OpenAIEnrichmentProvider calls the OpenAI chat completions API to produce
// structured resource enrichment. It is only used when enabled with an API key.
type OpenAIEnrichmentProvider struct {
	settings ProviderSettings
	client   *http.Client
}

// NewOpenAIEnrichmentProvider builds an enrichment provider. The model defaults
// to gpt-4o-mini when unset.
func NewOpenAIEnrichmentProvider(settings ProviderSettings) *OpenAIEnrichmentProvider {
	if settings.Model == "" {
		settings.Model = "gpt-4o-mini"
	}
	if settings.BaseURL == "" {
		settings.BaseURL = "https://api.openai.com"
	}
	if settings.TimeoutSeconds <= 0 {
		settings.TimeoutSeconds = 30
	}
	return &OpenAIEnrichmentProvider{
		settings: settings,
		client:   &http.Client{Timeout: time.Duration(settings.TimeoutSeconds) * time.Second},
	}
}

func (p *OpenAIEnrichmentProvider) Name() string { return "openai-enrichment" }

func (p *OpenAIEnrichmentProvider) Enrich(ctx context.Context, input EnrichmentInput) (EnrichmentResult, error) {
	if !p.settings.Enabled || strings.TrimSpace(p.settings.APIKey) == "" {
		return EnrichmentResult{}, ErrProviderUnavailable
	}

	prompt := buildEnrichmentPrompt(input)
	endpoint := strings.TrimRight(p.settings.BaseURL, "/") + "/v1/chat/completions"

	reqBody := map[string]any{
		"model": p.settings.Model,
		"messages": []map[string]string{
			{"role": "system", "content": "You are a strict JSON enrichment service. Output valid JSON only."},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.2,
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return EnrichmentResult{}, fmt.Errorf("marshal enrichment request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return EnrichmentResult{}, fmt.Errorf("build enrichment request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.settings.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return EnrichmentResult{}, fmt.Errorf("enrichment request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return EnrichmentResult{}, fmt.Errorf("read enrichment response: %w", err)
	}
	if resp.StatusCode >= 300 {
		return EnrichmentResult{}, fmt.Errorf("enrichment status %d: %s", resp.StatusCode, string(body))
	}

	var decoded struct {
		Choices []struct {
			Message struct{ Content string `json:"content"` } `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return EnrichmentResult{}, fmt.Errorf("decode enrichment response: %w", err)
	}
	if len(decoded.Choices) == 0 {
		return EnrichmentResult{}, fmt.Errorf("enrichment returned no choices")
	}

	return parseEnrichmentOutput(decoded.Choices[0].Message.Content)
}

func buildEnrichmentPrompt(input EnrichmentInput) string {
	content := strings.TrimSpace(input.Content)
	if len([]rune(content)) > 3000 {
		content = string([]rune(content)[:3000])
	}
	return fmt.Sprintf(
		"Summarise this resource and extract key points and named entities.\n"+
			`Return strict JSON only: {"summary":"...","key_points":["..."],"entities":["..."]}.`+"\n"+
			"Title: %s\nURL: %s\nContent: %s",
		input.Title, input.URL, content,
	)
}

func parseEnrichmentOutput(raw string) (EnrichmentResult, error) {
	jsonPayload := extractJSONObject(raw)
	if jsonPayload == "" {
		return EnrichmentResult{}, fmt.Errorf("enrichment response contained no JSON object")
	}

	var parsed struct {
		Summary   string   `json:"summary"`
		KeyPoints []string `json:"key_points"`
		Entities  []string `json:"entities"`
	}
	if err := json.Unmarshal([]byte(jsonPayload), &parsed); err != nil {
		return EnrichmentResult{}, fmt.Errorf("parse enrichment json: %w", err)
	}

	return EnrichmentResult{
		Summary:   strings.TrimSpace(parsed.Summary),
		KeyPoints: cleanStrings(parsed.KeyPoints),
		Entities:  cleanStrings(parsed.Entities),
	}, nil
}

func cleanStrings(ss []string) []string {
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		if t := strings.TrimSpace(s); t != "" {
			out = append(out, t)
		}
	}
	return out
}
