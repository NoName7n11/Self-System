package service

import (
	"context"
	"testing"

	"selfsystems/internal/domain"
)

func TestNormalizeURL(t *testing.T) {
	normalized, host, err := normalizeURL("https://www.example.com/path")
	if err != nil {
		t.Fatalf("expected valid URL, got error: %v", err)
	}
	if host != "example.com" {
		t.Fatalf("expected host example.com, got %q", host)
	}
	if normalized == "" {
		t.Fatalf("expected non-empty normalized URL")
	}
}

func TestNormalizeURLRejectsInvalid(t *testing.T) {
	_, _, err := normalizeURL("not-a-url")
	if err == nil {
		t.Fatalf("expected error for invalid url")
	}
}

func TestSemanticScoreRewardsRelevantResource(t *testing.T) {
	query := "knowledge graph ai"
	queryTokens := semanticExpandTokens(semanticTokenize(query))

	relevant := domain.Resource{
		Title:        "Designing AI Knowledge Graph Systems",
		Summary:      "This article explains graph relationships for machine learning agents",
		URL:          "https://example.com/ai-graph",
		CategoryName: "AI",
	}
	unrelated := domain.Resource{
		Title:        "Cooking Pasta Basics",
		Summary:      "Simple recipes for dinner",
		URL:          "https://example.com/cooking",
		CategoryName: "Food",
	}

	relevantScore := semanticScore(query, queryTokens, relevant)
	unrelatedScore := semanticScore(query, queryTokens, unrelated)

	if relevantScore <= unrelatedScore {
		t.Fatalf("expected relevant score to be higher: relevant=%f unrelated=%f", relevantScore, unrelatedScore)
	}
	if relevantScore <= 0 {
		t.Fatalf("expected relevant score > 0")
	}
}

func TestSemanticSearchEmptyQuery(t *testing.T) {
	service := &ResourceService{}
	results, err := service.SemanticSearch(context.Background(), "   ", 10)
	if err != nil {
		t.Fatalf("expected no error for empty query, got: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected no results for empty query")
	}
}
