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

// OpenAIEmbeddingProvider calls the OpenAI embeddings API. It is only used when
// enabled with an API key; otherwise it reports ErrProviderUnavailable so the
// manager falls back to the local embedder.
type OpenAIEmbeddingProvider struct {
	settings ProviderSettings
	client   *http.Client
}

// NewOpenAIEmbeddingProvider builds an embedding provider. The model defaults to
// text-embedding-3-small when unset.
func NewOpenAIEmbeddingProvider(settings ProviderSettings) *OpenAIEmbeddingProvider {
	if settings.Model == "" {
		settings.Model = "text-embedding-3-small"
	}
	if settings.BaseURL == "" {
		settings.BaseURL = "https://api.openai.com"
	}
	if settings.TimeoutSeconds <= 0 {
		settings.TimeoutSeconds = 20
	}
	return &OpenAIEmbeddingProvider{
		settings: settings,
		client:   &http.Client{Timeout: time.Duration(settings.TimeoutSeconds) * time.Second},
	}
}

func (p *OpenAIEmbeddingProvider) Name() string { return "openai-embedding" }

func (p *OpenAIEmbeddingProvider) GenerateEmbedding(ctx context.Context, text string) (Embedding, error) {
	if !p.settings.Enabled || strings.TrimSpace(p.settings.APIKey) == "" {
		return Embedding{}, ErrProviderUnavailable
	}

	endpoint := strings.TrimRight(p.settings.BaseURL, "/") + "/v1/embeddings"
	reqBody := map[string]any{
		"model": p.settings.Model,
		"input": text,
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return Embedding{}, fmt.Errorf("marshal embedding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return Embedding{}, fmt.Errorf("build embedding request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.settings.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return Embedding{}, fmt.Errorf("embedding request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Embedding{}, fmt.Errorf("read embedding response: %w", err)
	}
	if resp.StatusCode >= 300 {
		return Embedding{}, fmt.Errorf("embedding status %d: %s", resp.StatusCode, string(body))
	}

	var decoded struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return Embedding{}, fmt.Errorf("decode embedding response: %w", err)
	}
	if len(decoded.Data) == 0 || len(decoded.Data[0].Embedding) == 0 {
		return Embedding{}, fmt.Errorf("embedding response contained no vector")
	}

	vec := decoded.Data[0].Embedding
	return Embedding{
		Vector:       vec,
		ModelVersion: "openai:" + p.settings.Model,
		Dim:          len(vec),
	}, nil
}
