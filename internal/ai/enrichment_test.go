package ai

import (
	"context"
	"testing"
)

type fakeEnrichmentProvider struct {
	name   string
	result EnrichmentResult
	err    error
}

func (p fakeEnrichmentProvider) Name() string { return p.name }
func (p fakeEnrichmentProvider) Enrich(_ context.Context, _ EnrichmentInput) (EnrichmentResult, error) {
	if p.err != nil {
		return EnrichmentResult{}, p.err
	}
	return p.result, nil
}

func TestManager_EnrichResource_Success(t *testing.T) {
	mgr := NewManager("heuristic")
	mgr.RegisterEnrichment(fakeEnrichmentProvider{
		name: "fake",
		result: EnrichmentResult{
			Summary:   "A summary of the content",
			KeyPoints: []string{"point one", "point two"},
			Entities:  []string{"AI", "Go"},
		},
	})

	result, err := mgr.EnrichResource(context.Background(), EnrichmentInput{
		Title:   "Test Resource",
		URL:     "https://example.com",
		Content: "Some content",
	})
	if err != nil {
		t.Fatalf("EnrichResource: %v", err)
	}
	if result.Summary != "A summary of the content" {
		t.Errorf("Summary = %q, want %q", result.Summary, "A summary of the content")
	}
	if len(result.KeyPoints) != 2 {
		t.Errorf("KeyPoints len = %d, want 2", len(result.KeyPoints))
	}
	if result.Provider != "fake" {
		t.Errorf("Provider = %q, want fake", result.Provider)
	}
}

func TestManager_EnrichResource_SkipsUnavailable(t *testing.T) {
	mgr := NewManager("heuristic")
	mgr.RegisterEnrichment(fakeEnrichmentProvider{name: "unavailable", err: ErrProviderUnavailable})
	mgr.RegisterEnrichment(fakeEnrichmentProvider{
		name:   "fallback",
		result: EnrichmentResult{Summary: "fallback summary"},
	})

	result, err := mgr.EnrichResource(context.Background(), EnrichmentInput{Content: "text"})
	if err != nil {
		t.Fatalf("EnrichResource: %v", err)
	}
	if result.Provider != "fallback" {
		t.Errorf("Provider = %q, want fallback", result.Provider)
	}
}

func TestManager_EnrichResource_NoProviders(t *testing.T) {
	mgr := NewManager("heuristic")
	_, err := mgr.EnrichResource(context.Background(), EnrichmentInput{Content: "text"})
	if err == nil {
		t.Error("expected error when no enrichment providers registered")
	}
}

func TestParseEnrichmentOutput_ValidJSON(t *testing.T) {
	raw := `{"summary":"A good summary","key_points":["point A","point B"],"entities":["Entity1"]}`
	result, err := parseEnrichmentOutput(raw)
	if err != nil {
		t.Fatalf("parseEnrichmentOutput: %v", err)
	}
	if result.Summary != "A good summary" {
		t.Errorf("Summary = %q", result.Summary)
	}
	if len(result.KeyPoints) != 2 {
		t.Errorf("KeyPoints = %v", result.KeyPoints)
	}
	if len(result.Entities) != 1 || result.Entities[0] != "Entity1" {
		t.Errorf("Entities = %v", result.Entities)
	}
}

func TestParseEnrichmentOutput_JsonWrappedInText(t *testing.T) {
	// Model may prefix JSON with text — extractJSONObject should strip it.
	raw := `Here is the result: {"summary":"wrapped","key_points":[],"entities":[]}`
	result, err := parseEnrichmentOutput(raw)
	if err != nil {
		t.Fatalf("parseEnrichmentOutput: %v", err)
	}
	if result.Summary != "wrapped" {
		t.Errorf("Summary = %q, want wrapped", result.Summary)
	}
}

func TestParseEnrichmentOutput_NoJSON(t *testing.T) {
	_, err := parseEnrichmentOutput("no json here")
	if err == nil {
		t.Error("expected error for response with no JSON object")
	}
}
