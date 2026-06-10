package ai

import (
	"context"
	"errors"
	"testing"
)

type stubProvider struct {
	name   string
	output ClassificationOutput
	err    error
}

func (p stubProvider) Name() string {
	return p.name
}

func (p stubProvider) ClassifySkim(_ context.Context, _ ClassificationInput) (ClassificationOutput, error) {
	if p.err != nil {
		return ClassificationOutput{}, p.err
	}
	return p.output, nil
}

func TestManagerFallsBackToHeuristic(t *testing.T) {
	manager := NewManager("openai")
	manager.Register(stubProvider{name: "openai", err: ErrProviderUnavailable})
	manager.Register(stubProvider{name: "heuristic", output: ClassificationOutput{SuggestedCategory: "Research", Confidence: 0.7}})
	manager.SetFallback("heuristic")

	result, err := manager.ClassifySkim(context.Background(), ClassificationInput{URL: "https://example.com/research"})
	if err != nil {
		t.Fatalf("expected fallback classification to succeed, got error: %v", err)
	}
	if result.SuggestedCategory != "Research" {
		t.Fatalf("expected Research, got %q", result.SuggestedCategory)
	}
}

func TestManagerReturnsPrimaryOnSuccess(t *testing.T) {
	manager := NewManager("gemini")
	manager.Register(stubProvider{name: "gemini", output: ClassificationOutput{SuggestedCategory: "AI", Confidence: 0.88}})
	manager.Register(stubProvider{name: "heuristic", output: ClassificationOutput{SuggestedCategory: "General", Confidence: 0.6}})
	manager.SetFallback("heuristic")

	result, err := manager.ClassifySkim(context.Background(), ClassificationInput{URL: "https://example.com/ai"})
	if err != nil {
		t.Fatalf("expected primary classification to succeed, got error: %v", err)
	}
	if result.SuggestedCategory != "AI" {
		t.Fatalf("expected AI, got %q", result.SuggestedCategory)
	}
}

func TestManagerErrorsWhenAllProvidersFail(t *testing.T) {
	manager := NewManager("openai")
	manager.Register(stubProvider{name: "openai", err: errors.New("boom")})
	manager.Register(stubProvider{name: "heuristic", err: ErrProviderUnavailable})
	manager.SetFallback("heuristic")

	_, err := manager.ClassifySkim(context.Background(), ClassificationInput{URL: "https://example.com"})
	if err == nil {
		t.Fatalf("expected error when providers fail")
	}
}
