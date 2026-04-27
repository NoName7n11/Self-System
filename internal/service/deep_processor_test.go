package service

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"selfsystems/internal/ai"
	sqliterepo "selfsystems/internal/repository/sqlite"
)

func TestDeepProcessorEnqueueAndProcessResource(t *testing.T) {
	resourceSvc, categoryRepo, categorySvc, manager, cleanup := newDeepProcessorTestFixture(t)
	defer cleanup()

	created, err := resourceSvc.Create(context.Background(), CreateResourceInput{
		URL:          "https://example.com/deep-processing",
		Title:        "Deep Processing Activation",
		Summary:      "Initial skim summary",
		CategoryName: "Research",
	})
	if err != nil {
		t.Fatalf("create resource: %v", err)
	}

	processor := NewDeepProcessor(resourceSvc, categoryRepo, categorySvc, manager, DeepProcessingSettings{
		Enabled:                 true,
		QueueCapacity:           16,
		WorkerCount:             1,
		MaxTasksPerMinute:       120,
		MaxTokensPerDay:         10000,
		ComplexityThreshold:     5,
		LowCostEstimatedTokens:  50,
		HighCostEstimatedTokens: 200,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	processor.Start(ctx)

	if err := processor.Enqueue(DeepTask{ResourceID: created.ID}); err != nil {
		t.Fatalf("enqueue deep task: %v", err)
	}

	waitForDeepCondition(t, 3*time.Second, func() bool {
		resource, getErr := resourceSvc.GetByID(context.Background(), created.ID)
		if getErr != nil || resource == nil {
			return false
		}
		return strings.Contains(resource.Summary, deepSummaryMarker)
	})

	metrics := processor.Metrics()
	if metrics.EnqueuedTotal < 1 {
		t.Fatalf("expected enqueued total >= 1, got %d", metrics.EnqueuedTotal)
	}
	if metrics.ProcessedTotal < 1 {
		t.Fatalf("expected processed total >= 1, got %d", metrics.ProcessedTotal)
	}
	if metrics.TokensUsedToday <= 0 {
		t.Fatalf("expected tokens used today > 0, got %d", metrics.TokensUsedToday)
	}
}

func TestDeepProcessorBudgetRejectsExcessWork(t *testing.T) {
	resourceSvc, categoryRepo, categorySvc, manager, cleanup := newDeepProcessorTestFixture(t)
	defer cleanup()

	first, err := resourceSvc.Create(context.Background(), CreateResourceInput{
		URL:          "https://example.com/deep-budget-first",
		Title:        "Budget First",
		CategoryName: "Research",
	})
	if err != nil {
		t.Fatalf("create first resource: %v", err)
	}

	second, err := resourceSvc.Create(context.Background(), CreateResourceInput{
		URL:          "https://example.com/deep-budget-second",
		Title:        "Budget Second",
		CategoryName: "Research",
	})
	if err != nil {
		t.Fatalf("create second resource: %v", err)
	}

	processor := NewDeepProcessor(resourceSvc, categoryRepo, categorySvc, manager, DeepProcessingSettings{
		Enabled:                 true,
		QueueCapacity:           16,
		WorkerCount:             1,
		MaxTasksPerMinute:       120,
		MaxTokensPerDay:         220,
		ComplexityThreshold:     999,
		LowCostEstimatedTokens:  150,
		HighCostEstimatedTokens: 400,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	processor.Start(ctx)

	if err := processor.Enqueue(DeepTask{ResourceID: first.ID}); err != nil {
		t.Fatalf("enqueue first deep task: %v", err)
	}
	if err := processor.Enqueue(DeepTask{ResourceID: second.ID}); err != nil {
		t.Fatalf("enqueue second deep task: %v", err)
	}

	waitForDeepCondition(t, 3*time.Second, func() bool {
		metrics := processor.Metrics()
		return metrics.ProcessedTotal+metrics.FailedTotal >= 2
	})

	metrics := processor.Metrics()
	if metrics.ProcessedTotal != 1 {
		t.Fatalf("expected exactly one processed task under token budget, got %d", metrics.ProcessedTotal)
	}
	if metrics.RejectedTotal < 1 {
		t.Fatalf("expected rejected total >= 1 when token budget exceeded, got %d", metrics.RejectedTotal)
	}
	if metrics.TokensUsedToday > int64(metrics.MaxTokensPerDay) {
		t.Fatalf("expected tokens used to remain within budget, got used=%d budget=%d", metrics.TokensUsedToday, metrics.MaxTokensPerDay)
	}
}

func TestDeepProcessorReturnsDisabledErrorWhenOff(t *testing.T) {
	processor := NewDeepProcessor(nil, nil, nil, nil, DeepProcessingSettings{Enabled: false})
	if err := processor.Enqueue(DeepTask{ResourceID: "res-1"}); err == nil {
		t.Fatalf("expected enqueue error when deep processing is disabled")
	} else if err != ErrDeepProcessingDisabled {
		t.Fatalf("expected ErrDeepProcessingDisabled, got %v", err)
	}
}

func newDeepProcessorTestFixture(t *testing.T) (*ResourceService, *sqliterepo.CategoryRepository, *CategoryService, *ai.Manager, func()) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "deep_processor_test.db")
	db, err := sqliterepo.Open(dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	categoryRepo := sqliterepo.NewCategoryRepository(db)
	resourceRepo := sqliterepo.NewResourceRepository(db)

	manager := ai.NewManager("heuristic")
	heuristic := ai.NewHeuristicProvider()
	manager.Register(heuristic)
	manager.SetFallback(heuristic.Name())

	categorySvc := NewCategoryService(categoryRepo)
	classifier := NewCategoryClassifier(categoryRepo, manager)
	resourceSvc := NewResourceService(resourceRepo, categoryRepo, classifier, categorySvc)

	cleanup := func() {
		_ = db.Close()
	}

	return resourceSvc, categoryRepo, categorySvc, manager, cleanup
}

func waitForDeepCondition(t *testing.T, timeout time.Duration, predicate func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if predicate() {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}

	t.Fatalf("condition not satisfied within %s", timeout)
}
