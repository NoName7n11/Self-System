package service

import (
	"path/filepath"
	"testing"

	"selfsystems/internal/domain"
)

func TestDeepComplexityScoreUsesHostnameNotPathKeywords(t *testing.T) {
	base := domain.Resource{
		Title:   "Short title",
		Summary: "Short summary",
		URL:     "https://evil.com/path/without/keywords",
	}
	withPathKeywords := base
	withPathKeywords.URL = "https://evil.com/github/docs/research/item"

	scoreBase := deepComplexityScore(base)
	scorePathKeywords := deepComplexityScore(withPathKeywords)
	if scorePathKeywords != scoreBase {
		t.Fatalf("expected path keywords to not affect score, got base=%d path=%d", scoreBase, scorePathKeywords)
	}

	withHostKeyword := base
	withHostKeyword.URL = "https://docs.example.com/item"
	scoreHostKeyword := deepComplexityScore(withHostKeyword)
	if scoreHostKeyword <= scoreBase {
		t.Fatalf("expected hostname keyword to increase score, got base=%d host=%d", scoreBase, scoreHostKeyword)
	}
}

func TestDeepProcessorPersistsDailyBudgetState(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "deep-budget.json")

	processor := NewDeepProcessor(nil, nil, nil, nil, DeepProcessingSettings{
		Enabled:                 true,
		QueueCapacity:           1,
		WorkerCount:             1,
		MaxTokensPerDay:         10,
		LowCostEstimatedTokens:  1,
		HighCostEstimatedTokens: 1,
		BudgetStatePath:         statePath,
	})

	if !processor.reserveTokenBudget(3) {
		t.Fatalf("expected initial budget reservation to succeed")
	}

	reloaded := NewDeepProcessor(nil, nil, nil, nil, DeepProcessingSettings{
		Enabled:                 true,
		QueueCapacity:           1,
		WorkerCount:             1,
		MaxTokensPerDay:         10,
		LowCostEstimatedTokens:  1,
		HighCostEstimatedTokens: 1,
		BudgetStatePath:         statePath,
	})

	if reloaded.tokensUsedToday.Load() != 3 {
		t.Fatalf("expected persisted tokens_used_today 3, got %d", reloaded.tokensUsedToday.Load())
	}
}
