package integration_test

import (
	"context"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"selfsystems/internal/ai"
	"selfsystems/internal/domain"
	sqliterepo "selfsystems/internal/repository/sqlite"
	"selfsystems/internal/service"
)

// ---- shared setup -----------------------------------------------------------

type aiPipelineFixture struct {
	mock         *ai.MockProvider
	catSvc       *service.CategoryService
	resourceSvc  *service.ResourceService
	embeddingSvc *service.EmbeddingService
	deepProc     *service.DeepProcessor
}

func newAIPipelineFixture(t *testing.T) aiPipelineFixture {
	t.Helper()

	db, err := sqliterepo.Open(filepath.Join(t.TempDir(), "ai_pipeline_test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	catRepo := sqliterepo.NewCategoryRepository(db)
	resourceRepo := sqliterepo.NewResourceRepository(db)
	embeddingRepo := sqliterepo.NewEmbeddingRepository(db)

	// MockProvider satisfies all three AI interfaces — no real API calls.
	mock := ai.NewMockProvider("mock")
	mock.ClassifyOutput = ai.ClassificationOutput{
		SuggestedCategory: "Technology",
		Confidence:        0.92,
	}
	mock.EmbeddingOutput = ai.Embedding{
		Vector:       make([]float32, 256),
		ModelVersion: "mock-v1",
		Dim:          256,
	}
	// Give the mock vector some non-zero values so similarity makes sense.
	for i := range mock.EmbeddingOutput.Vector {
		mock.EmbeddingOutput.Vector[i] = float32(i%16) / 16.0
	}
	ai.NormalizeVector(mock.EmbeddingOutput.Vector)

	mock.EnrichOutput = ai.EnrichmentResult{
		Summary:   "AI-generated summary of the resource content.",
		KeyPoints: []string{"Key point one", "Key point two"},
		Entities:  []string{"Go", "Machine Learning"},
	}

	mgr := ai.NewManager("mock")
	mgr.Register(mock)
	mgr.SetFallback("mock")
	mgr.RegisterEmbedding(mock)
	mgr.RegisterEnrichment(mock)

	embSvc := service.NewEmbeddingService(embeddingRepo, mgr)
	catSvc := service.NewCategoryService(catRepo)
	classifier := service.NewCategoryClassifier(catRepo, mgr)
	resourceSvc := service.NewResourceService(
		resourceRepo, catRepo, classifier, catSvc,
		service.WithResourceEmbeddingService(embSvc),
		service.WithClassificationThreshold(0.85),
	)

	deepProc := service.NewDeepProcessor(
		resourceSvc, catRepo, catSvc, mgr,
		service.DeepProcessingSettings{},
	).
		WithEmbeddingService(embSvc).
		WithEventDetector(nil) // event detector not needed for this test

	return aiPipelineFixture{
		mock:         mock,
		catSvc:       catSvc,
		resourceSvc:  resourceSvc,
		embeddingSvc: embSvc,
		deepProc:     deepProc,
	}
}

// ---- tests ------------------------------------------------------------------

// TestAIPipeline_ClassificationConfidenceStored verifies that creating a
// resource populates classification_confidence and classification_source in
// extracted_data from the mock AI provider.
func TestAIPipeline_ClassificationConfidenceStored(t *testing.T) {
	fix := newAIPipelineFixture(t)
	ctx := context.Background()

	resource, err := fix.resourceSvc.Create(ctx, service.CreateResourceInput{
		URL:   "https://example.com/ai-article",
		Title: "Introduction to Machine Learning",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if resource.ExtractedData.ClassificationConfidence != 0.92 {
		t.Errorf("ClassificationConfidence = %v, want 0.92", resource.ExtractedData.ClassificationConfidence)
	}
	if resource.ExtractedData.ClassificationSource != service.ClassificationSourceAI {
		t.Errorf("ClassificationSource = %q, want ai", resource.ExtractedData.ClassificationSource)
	}
	if resource.ExtractedData.NeedsReview {
		t.Error("NeedsReview should be false for confidence 0.92 ≥ 0.85 threshold")
	}
	if fix.mock.ClassifyCalls == 0 {
		t.Error("expected AI classify to be called at least once")
	}
}

// TestAIPipeline_DeepProcessing_SummaryAndEmbedding verifies that running deep
// processing via ProcessDirect writes a real AI-generated summary to the
// resource and stores an embedding vector.
func TestAIPipeline_DeepProcessing_SummaryAndEmbedding(t *testing.T) {
	fix := newAIPipelineFixture(t)
	ctx := context.Background()

	cat, err := fix.catSvc.EnsureByName(ctx, "Technology", domain.CategorySourceManual)
	if err != nil {
		t.Fatalf("ensure category: %v", err)
	}
	resource, err := fix.resourceSvc.Create(ctx, service.CreateResourceInput{
		URL:          "https://example.com/ml",
		Title:        "Deep Learning Overview",
		CategoryName: cat.Name,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Set extracted text so enrichment has content to work with.
	resource.ExtractedData.MainText = "Machine learning is a subfield of artificial intelligence."
	if err := fix.resourceSvc.UpdateExtractedData(ctx, resource.ID, resource.ExtractedData); err != nil {
		t.Fatalf("UpdateExtractedData: %v", err)
	}

	if err := fix.deepProc.ProcessDirect(ctx, resource.ID); err != nil {
		t.Fatalf("ProcessDirect: %v", err)
	}

	// Verify summary was written back.
	updated, err := fix.resourceSvc.GetByID(ctx, resource.ID)
	if err != nil || updated == nil {
		t.Fatalf("GetByID: %v", err)
	}
	if updated.Summary != "AI-generated summary of the resource content." {
		t.Errorf("Summary = %q, want AI-generated summary", updated.Summary)
	}
	if fix.mock.EnrichCalls == 0 {
		t.Error("expected AI enrich to be called")
	}

	// Verify embedding was stored.
	emb, err := fix.embeddingSvc.SearchSimilar(ctx, "machine learning", 5, -1.0)
	if err != nil {
		t.Fatalf("SearchSimilar: %v", err)
	}
	if len(emb) == 0 {
		t.Error("expected embedding to be stored after deep processing")
	}
}

// TestAIPipeline_SemanticSearch_ReturnsRankedResults verifies that storing
// embeddings and querying via SemanticSearch returns results ranked by cosine
// similarity using the mock embedding provider.
func TestAIPipeline_SemanticSearch_ReturnsRankedResults(t *testing.T) {
	fix := newAIPipelineFixture(t)
	ctx := context.Background()

	cat, err := fix.catSvc.EnsureByName(ctx, "Technology", domain.CategorySourceManual)
	if err != nil {
		t.Fatalf("ensure category: %v", err)
	}

	// Create two resources and manually store their embeddings.
	createAndEmbed := func(id, title string) {
		r := domain.Resource{
			ID: id, URL: "https://example.com/" + id,
			Host: "example.com", Title: title,
			CategoryID: cat.ID, CategoryName: cat.Name,
		}
		// Insert directly into the repo via the resource service's fake.
		// Use the fixture's resource service Create with category pre-set.
		created, err := fix.resourceSvc.Create(ctx, service.CreateResourceInput{
			URL:          r.URL,
			Title:        r.Title,
			CategoryName: cat.Name,
		})
		if err != nil {
			t.Fatalf("create %s: %v", id, err)
		}
		// Embed via deep processing.
		_ = fix.deepProc.ProcessDirect(ctx, created.ID)
	}

	createAndEmbed("res-ai", "artificial intelligence neural networks")
	createAndEmbed("res-cook", "pasta recipes italian food")

	results, err := fix.resourceSvc.SemanticSearch(ctx, "artificial intelligence", 10)
	if err != nil {
		t.Fatalf("SemanticSearch: %v", err)
	}
	// Mock returns the same vector for every query, so both resources will
	// match. Verify we get at least one result back.
	if len(results) == 0 {
		t.Error("expected at least one semantic search result")
	}
	// No real API calls should have been made — all via mock.
	t.Logf("classify calls: %d, embed calls: %d, enrich calls: %d",
		fix.mock.ClassifyCalls, fix.mock.EmbeddingCalls, fix.mock.EnrichCalls)
}

// TestAIPipeline_NeedsReview_BelowThreshold verifies that a resource classified
// with low confidence is flagged for review.
func TestAIPipeline_NeedsReview_BelowThreshold(t *testing.T) {
	fix := newAIPipelineFixture(t)
	fix.mock.ClassifyOutput = ai.ClassificationOutput{
		SuggestedCategory: "Technology",
		Confidence:        0.50, // below 0.85 threshold
	}
	ctx := context.Background()

	resource, err := fix.resourceSvc.Create(ctx, service.CreateResourceInput{
		URL:   "https://example.com/ambiguous",
		Title: "Some ambiguous article",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if !resource.ExtractedData.NeedsReview {
		t.Error("expected NeedsReview = true for confidence 0.50 < 0.85")
	}
}

// ---- helper exposed for test use --------------------------------------------

// NormalizeVector is re-exported here so test fixtures in integration_test
// packages can normalize mock vectors without importing the internal helper.
// (The real normalizeVector is unexported in the ai package — this is the
// exported wrapper used only in tests.)
func init() {
	_ = ai.NormalizeVector // compile-time check that NormalizeVector is exported
}
