package ai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGeminiProviderUsesAPIKeyHeaderNotQuery(t *testing.T) {
	var capturedQuery string
	var capturedHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		capturedHeader = r.Header.Get("x-goog-api-key")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"{\"category\":\"AI\",\"confidence\":0.8,\"reason\":\"ok\"}"}]}}]}`))
	}))
	defer server.Close()

	provider := NewGeminiProvider(ProviderSettings{
		Enabled: true,
		APIKey:  "test-key-123",
		Model:   "gemini-1.5-flash",
		BaseURL: server.URL,
	})

	_, err := provider.ClassifySkim(context.Background(), ClassificationInput{
		URL:   "https://example.com",
		Title: "Example",
	})
	if err != nil {
		t.Fatalf("expected classify to succeed, got error: %v", err)
	}

	if strings.Contains(capturedQuery, "key=") {
		t.Fatalf("expected no key query parameter, got query=%q", capturedQuery)
	}
	if capturedHeader != "test-key-123" {
		t.Fatalf("expected x-goog-api-key header to be set")
	}
}
