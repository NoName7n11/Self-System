package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"selfsystems/internal/domain"
	"selfsystems/internal/eventstore"
	"selfsystems/internal/extractor"
	"selfsystems/internal/gbus"
)

// SkimExtractor is the interface satisfied by *extractor.URLExtractor.
// Defined here so callers can inject a test double without importing the package.
type SkimExtractor interface {
	Extract(ctx context.Context, url string) (extractor.URLExtractResult, error)
}

type ResourceService struct {
	resources     domain.ResourceRepository
	categories    domain.CategoryRepository
	classifier    *CategoryClassifier
	catSvc        *CategoryService
	eventStore    eventstore.Store
	projectors    *eventstore.ProjectorRegistry
	eventsEnabled bool
	eventObs      *eventstore.EventObservability
	skimExtractor SkimExtractor
	// classificationThreshold is the minimum confidence for auto-save. Below this,
	// auto-classified resources are flagged NeedsReview. Default 0.85.
	classificationThreshold float64
	// embeddingSvc powers vector-backed semantic search when configured.
	embeddingSvc *EmbeddingService
	// gbusEmitter fires async GBUS interaction signals when configured.
	gbusEmitter *gbus.SignalEmitter
	// gbusInference applies learned category affinity scores when configured.
	gbusInference *gbus.Inference
}

// ResourceServiceOption configures a ResourceService.
type ResourceServiceOption func(*ResourceService)

// WithEventSourcing enables the event-sourced write path (P7) for resources.
// When enabled, mutations append events and project to the resources table
// inside the same transaction. When disabled (default), writes go directly
// to the state table.
func WithEventSourcing(store eventstore.Store, registry *eventstore.ProjectorRegistry) ResourceServiceOption {
	return func(s *ResourceService) {
		if store != nil && registry != nil {
			s.eventStore = store
			s.projectors = registry
			s.eventsEnabled = true
		}
	}
}

// WithResourceEventObservability wires an EventObservability into the service
// so that append and OCC retry counts are tracked (WS6 — observability).
func WithResourceEventObservability(obs *eventstore.EventObservability) ResourceServiceOption {
	return func(s *ResourceService) {
		s.eventObs = obs
	}
}

// WithSkimExtractor enables async URL extraction on resource creation.
// After Create returns, a background goroutine fetches the URL and populates
// extracted_data (title, description, page type, main text).
func WithSkimExtractor(e SkimExtractor) ResourceServiceOption {
	return func(s *ResourceService) {
		s.skimExtractor = e
	}
}

// WithResourceEmbeddingService wires the embedding service into the resource
// service so that SemanticSearch and HybridSearch can use vector similarity.
func WithResourceEmbeddingService(svc *EmbeddingService) ResourceServiceOption {
	return func(s *ResourceService) {
		s.embeddingSvc = svc
	}
}

// WithGBUSEmitter wires a GBUS signal emitter for interaction signal emission.
func WithGBUSEmitter(emitter *gbus.SignalEmitter) ResourceServiceOption {
	return func(s *ResourceService) {
		s.gbusEmitter = emitter
	}
}

// WithGBUSInference wires a GBUS inference engine to bias classification and
// rerank semantic search results by learned category affinity scores.
func WithGBUSInference(inf *gbus.Inference) ResourceServiceOption {
	return func(s *ResourceService) {
		s.gbusInference = inf
	}
}

// WithClassificationThreshold sets the minimum confidence for auto-save.
// Auto-classified resources scoring below this are flagged NeedsReview.
func WithClassificationThreshold(threshold float64) ResourceServiceOption {
	return func(s *ResourceService) {
		if threshold > 0 && threshold <= 1 {
			s.classificationThreshold = threshold
		}
	}
}

type CreateResourceInput struct {
	// ID is optional. When set (e.g. during offline replay) it is used as the
	// resource's primary key so the originating device's identity is preserved
	// across sync. When empty a new UUID is generated.
	ID           string
	URL          string
	Title        string
	Summary      string
	CategoryID   string
	CategoryName string
}

type UpdateResourceInput struct {
	ID           string
	URL          string
	Title        string
	Summary      string
	CategoryID   string
	CategoryName string
}

type UpdateResourceCategoryInput struct {
	ResourceID string
	CategoryID string
}

// EventsEnabled reports whether the event-sourced write path is active.
// When true, the HTTP handler must skip its direct hub.Publish calls for
// resources; the async projectors or outbox worker handle delivery instead.
func (s *ResourceService) EventsEnabled() bool {
	return s.eventsEnabled
}

func NewResourceService(
	resources domain.ResourceRepository,
	categories domain.CategoryRepository,
	classifier *CategoryClassifier,
	catSvc *CategoryService,
	opts ...ResourceServiceOption,
) *ResourceService {
	s := &ResourceService{
		resources:               resources,
		categories:              categories,
		classifier:              classifier,
		catSvc:                  catSvc,
		classificationThreshold: 0.85,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *ResourceService) Create(ctx context.Context, input CreateResourceInput) (domain.Resource, error) {
	normalizedURL, host, err := normalizeURL(input.URL)
	if err != nil {
		return domain.Resource{}, err
	}

	// Fast-path exact URL duplicate detection (WS1).
	// When the same URL already exists: increment counter, return existing resource.
	if existing, err := s.resources.FindByURL(ctx, normalizedURL); err == nil && existing != nil {
		if incErr := s.resources.IncrementCounter(ctx, existing.ID); incErr != nil {
			slog.Warn("duplicate resource: failed to increment counter", "resource_id", existing.ID, "error", incErr)
		} else {
			existing.SaveCount++
		}
		if s.eventsEnabled {
			_ = s.emitCounterIncrementedEvent(ctx, existing.ID, existing.SaveCount)
		}
		if s.gbusEmitter != nil {
			s.gbusEmitter.Emit(ctx, gbus.GBUSSignalPayload{
				SignalType: gbus.SignalCounterIncremented,
				ResourceID: existing.ID,
				CategoryID: existing.CategoryID,
			})
		}
		return *existing, domain.ErrDuplicateResource
	}

	var category domain.Category
	userOverride := false
	classificationConfidence := 1.0
	classificationSource := ClassificationSourceUser

	if strings.TrimSpace(input.CategoryID) != "" {
		categoryPtr, err := s.categories.GetByID(ctx, strings.TrimSpace(input.CategoryID))
		if err != nil {
			return domain.Resource{}, err
		}
		if categoryPtr == nil {
			return domain.Resource{}, fmt.Errorf("category not found")
		}
		category = *categoryPtr
		userOverride = true
	} else if strings.TrimSpace(input.CategoryName) != "" {
		category, err = s.catSvc.EnsureByName(ctx, input.CategoryName, domain.CategorySourceManual)
		if err != nil {
			return domain.Resource{}, err
		}
		userOverride = true
	} else {
		suggestion, err := s.classifier.Suggest(ctx, normalizedURL, input.Title)
		if err != nil {
			return domain.Resource{}, err
		}
		category = suggestion.Category
		classificationConfidence = suggestion.Score
		classificationSource = suggestion.Source
		if classificationSource == "" {
			classificationSource = ClassificationSourceHeuristic
		}
		// Apply GBUS affinity bias when inference is available and confidence is
		// below the review threshold (so the boost can push marginal cases over).
		if s.gbusInference != nil && classificationConfidence < s.classificationThreshold {
			classificationConfidence = s.gbusInference.BiasClassification(category.ID, classificationConfidence)
		}
	}

	// Apply the confidence threshold: auto-classified resources scoring below
	// the threshold are flagged for review. Manual assignments never need review.
	needsReview := !userOverride && classificationConfidence < s.classificationThreshold

	now := time.Now().UTC()
	resourceID := uuid.NewString()
	if strings.TrimSpace(input.ID) != "" {
		resourceID = strings.TrimSpace(input.ID)
	}
	resource := domain.Resource{
		ID:           resourceID,
		URL:          normalizedURL,
		Host:         host,
		Title:        strings.TrimSpace(input.Title),
		Summary:      strings.TrimSpace(input.Summary),
		CategoryID:   category.ID,
		CategoryName: category.Name,
		UserOverride: userOverride,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	resource.ExtractedData.ClassificationConfidence = classificationConfidence
	resource.ExtractedData.ClassificationSource = classificationSource
	resource.ExtractedData.NeedsReview = needsReview

	if resource.Title == "" {
		resource.Title = inferTitleFromURL(normalizedURL)
	}

	if s.eventsEnabled {
		if err := s.createWithEvents(ctx, &resource); err != nil {
			return domain.Resource{}, err
		}
	} else {
		if err := s.resources.Create(ctx, &resource); err != nil {
			return domain.Resource{}, err
		}
	}

	if err := s.categories.IncrementAccept(ctx, category.ID); err != nil {
		return domain.Resource{}, err
	}

	// Emit GBUS signal for classification source.
	if s.gbusEmitter != nil {
		signalType := gbus.SignalAutoClassification
		if userOverride {
			signalType = gbus.SignalManualClassification
		}
		s.gbusEmitter.Emit(ctx, gbus.GBUSSignalPayload{
			SignalType: signalType,
			ResourceID: resource.ID,
			CategoryID: resource.CategoryID,
		})
	}

	// Fire async skim extraction for URL resources (non-blocking).
	if s.skimExtractor != nil && normalizedURL != "" {
		titleWasInferred := strings.TrimSpace(input.Title) == ""
		go s.runSkimExtraction(resource, titleWasInferred)
	}

	return resource, nil
}

// runSkimExtraction fetches the resource URL and writes the extracted metadata
// back via UpdateExtractedData. It runs in a goroutine and never returns an
// error — failures are silently dropped so the resource remains usable.
func (s *ResourceService) runSkimExtraction(resource domain.Resource, titleWasInferred bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	result, err := s.skimExtractor.Extract(ctx, resource.URL)
	if err != nil {
		return
	}

	data := resource.ExtractedData
	data.ExtractedTitle = result.Title
	data.ExtractedDescription = result.Description
	data.PageType = result.PageType
	if result.MainText != "" {
		data.MainText = result.MainText
	}

	updateCtx, updateCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer updateCancel()

	if err := s.resources.UpdateExtractedData(updateCtx, resource.ID, data); err != nil {
		slog.Warn("skim extractor: failed to write extracted_data", "resource_id", resource.ID, "error", err)
	}

	// If title was auto-inferred from URL and extraction found a real title,
	// promote the extracted title to the visible resource title.
	if titleWasInferred && result.Title != "" {
		resource.Title = result.Title
		resource.ExtractedData = data
		if err := s.resources.Update(updateCtx, &resource); err != nil {
			slog.Warn("skim extractor: failed to promote extracted title", "resource_id", resource.ID, "error", err)
		}
	}
}

// UpdateExtractedData writes extracted_data for a resource without going through
// the full domain mutation path. Called by the deep processing extraction workers.
func (s *ResourceService) UpdateExtractedData(ctx context.Context, resourceID string, data domain.ResourceExtractedData) error {
	return s.resources.UpdateExtractedData(ctx, resourceID, data)
}

func (s *ResourceService) List(ctx context.Context, limit, offset int) ([]domain.Resource, error) {
	return s.resources.List(ctx, limit, offset)
}

func (s *ResourceService) GetByID(ctx context.Context, id string) (*domain.Resource, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("resource id is required")
	}

	return s.resources.GetByID(ctx, id)
}

func (s *ResourceService) Update(ctx context.Context, input UpdateResourceInput) (*domain.Resource, error) {
	id := strings.TrimSpace(input.ID)
	if id == "" {
		return nil, fmt.Errorf("resource id is required")
	}

	existing, err := s.resources.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, nil
	}

	normalizedURL := existing.URL
	host := existing.Host
	if strings.TrimSpace(input.URL) != "" {
		normalizedURL, host, err = normalizeURL(input.URL)
		if err != nil {
			return nil, err
		}
	}

	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = inferTitleFromURL(normalizedURL)
	}

	summary := strings.TrimSpace(input.Summary)
	category := domain.Category{ID: existing.CategoryID, Name: existing.CategoryName}
	userOverride := existing.UserOverride
	oldCategoryID := existing.CategoryID

	if categoryID := strings.TrimSpace(input.CategoryID); categoryID != "" {
		if s.categories == nil {
			return nil, fmt.Errorf("category repository is not configured")
		}

		categoryPtr, getErr := s.categories.GetByID(ctx, categoryID)
		if getErr != nil {
			return nil, getErr
		}
		if categoryPtr == nil {
			return nil, fmt.Errorf("category not found")
		}
		category = *categoryPtr
		userOverride = true
	} else if categoryName := strings.TrimSpace(input.CategoryName); categoryName != "" {
		if s.catSvc == nil {
			return nil, fmt.Errorf("category service is not configured")
		}

		resolved, resolveErr := s.catSvc.EnsureByName(ctx, categoryName, domain.CategorySourceManual)
		if resolveErr != nil {
			return nil, resolveErr
		}
		category = resolved
		userOverride = true
	}

	now := time.Now().UTC()
	existing.URL = normalizedURL
	existing.Host = host
	existing.Title = title
	existing.Summary = summary
	existing.CategoryID = category.ID
	existing.CategoryName = category.Name
	existing.UserOverride = userOverride
	existing.UpdatedAt = now

	if s.eventsEnabled {
		if err := s.updateWithEvents(ctx, existing); err != nil {
			return nil, err
		}
	} else {
		if err := s.resources.Update(ctx, existing); err != nil {
			return nil, err
		}
	}

	if category.ID != "" && category.ID != oldCategoryID {
		if s.categories == nil {
			return nil, fmt.Errorf("category repository is not configured")
		}
		if err := s.categories.IncrementAccept(ctx, category.ID); err != nil {
			return nil, err
		}
	}

	return existing, nil
}

func (s *ResourceService) Delete(ctx context.Context, id string) (bool, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return false, fmt.Errorf("resource id is required")
	}

	existing, err := s.resources.GetByID(ctx, id)
	if err != nil {
		return false, err
	}
	if existing == nil {
		return false, nil
	}

	if s.eventsEnabled {
		if err := s.deleteWithEvents(ctx, id); err != nil {
			return false, err
		}
	} else {
		if err := s.resources.Delete(ctx, id); err != nil {
			return false, err
		}
	}

	if s.gbusEmitter != nil {
		s.gbusEmitter.Emit(ctx, gbus.GBUSSignalPayload{
			SignalType: gbus.SignalResourceDeleted,
			ResourceID: id,
			CategoryID: existing.CategoryID,
		})
	}

	return true, nil
}

func (s *ResourceService) Search(ctx context.Context, query string, limit int) ([]domain.Resource, error) {
	return s.resources.Search(ctx, query, limit)
}

// SemanticSearch returns resources ranked by vector similarity when an
// embedding service is configured, or falls back to token-based scoring.
func (s *ResourceService) SemanticSearch(ctx context.Context, query string, limit int) ([]domain.Resource, error) {
	trimmedQuery := strings.TrimSpace(query)
	if trimmedQuery == "" {
		return []domain.Resource{}, nil
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	if s.embeddingSvc != nil {
		results, err := s.vectorSearch(ctx, trimmedQuery, limit)
		if err == nil && len(results) > 0 {
			return results, nil
		}
		// Fall through to token search when vector search fails or returns nothing
		// (e.g. no embeddings have been generated yet for this database).
	}

	return s.tokenSearch(ctx, trimmedQuery, limit)
}

// vectorSearch uses the embedding service to find similar resources by cosine
// similarity, then fetches full resource objects and applies GBUS reranking.
func (s *ResourceService) vectorSearch(ctx context.Context, query string, limit int) ([]domain.Resource, error) {
	matches, err := s.embeddingSvc.SearchSimilar(ctx, query, limit, 0.0)
	if err != nil {
		return nil, err
	}
	results := make([]domain.Resource, 0, len(matches))
	for _, m := range matches {
		res, err := s.resources.GetByID(ctx, m.ResourceID)
		if err != nil || res == nil {
			continue
		}
		results = append(results, *res)
	}
	// Apply GBUS interest-based reranking when inference is configured.
	if s.gbusInference != nil && len(results) > 1 {
		ids := make([]string, len(results))
		catIDs := make([]string, len(results))
		for i, r := range results {
			ids[i] = r.ID
			catIDs[i] = r.CategoryID
		}
		perm := s.gbusInference.RerankByInterest(ids, catIDs, 0.5)
		reranked := make([]domain.Resource, len(results))
		for newIdx, oldIdx := range perm {
			reranked[newIdx] = results[oldIdx]
		}
		return reranked, nil
	}
	return results, nil
}

// tokenSearch is the legacy token-scoring fallback used when no embedding
// service is configured or when vector search fails.
func (s *ResourceService) tokenSearch(ctx context.Context, query string, limit int) ([]domain.Resource, error) {
	queryTokens := semanticExpandTokens(semanticTokenize(query))
	if len(queryTokens) == 0 {
		return []domain.Resource{}, nil
	}

	candidates, err := s.resources.List(ctx, 500, 0)
	if err != nil {
		return nil, err
	}

	type scoredResource struct {
		resource domain.Resource
		score    float64
	}

	scored := make([]scoredResource, 0, len(candidates))
	for _, candidate := range candidates {
		score := semanticScore(query, queryTokens, candidate)
		if score >= 0.08 {
			scored = append(scored, scoredResource{resource: candidate, score: score})
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return scored[i].resource.CreatedAt.After(scored[j].resource.CreatedAt)
		}
		return scored[i].score > scored[j].score
	})

	results := make([]domain.Resource, 0, limit)
	for idx, item := range scored {
		if idx >= limit {
			break
		}
		results = append(results, item.resource)
	}
	return results, nil
}

// HybridSearch merges keyword and semantic results using normalized rank scores.
// Semantic results carry weight 0.8; keyword results carry weight 1.0.
func (s *ResourceService) HybridSearch(ctx context.Context, query string, limit int) ([]domain.Resource, error) {
	trimmedQuery := strings.TrimSpace(query)
	if trimmedQuery == "" {
		return []domain.Resource{}, nil
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	fetch := limit * 3
	if fetch > 100 {
		fetch = 100
	}

	keyword, _ := s.Search(ctx, trimmedQuery, fetch)
	semantic, _ := s.SemanticSearch(ctx, trimmedQuery, fetch)

	return mergeHybridResults(keyword, semantic, limit), nil
}

// mergeHybridResults combines keyword and semantic result lists using normalized
// inverse-rank scoring. keyword weight = 1.0, semantic weight = 0.8.
func mergeHybridResults(keyword, semantic []domain.Resource, limit int) []domain.Resource {
	combined := map[string]float64{}
	order := map[string]domain.Resource{}

	addScores := func(items []domain.Resource, weight float64) {
		n := len(items)
		if n == 0 {
			return
		}
		for i, r := range items {
			rank := float64(n-i) / float64(n) // 1.0 → 1/n, descending
			combined[r.ID] += rank * weight
			if _, seen := order[r.ID]; !seen {
				order[r.ID] = r
			}
		}
	}

	addScores(keyword, 1.0)
	addScores(semantic, 0.8)

	type scored struct {
		res   domain.Resource
		score float64
	}
	all := make([]scored, 0, len(combined))
	for id, score := range combined {
		all = append(all, scored{res: order[id], score: score})
	}

	sort.Slice(all, func(i, j int) bool { return all[i].score > all[j].score })

	results := make([]domain.Resource, 0, limit)
	for _, item := range all {
		if len(results) >= limit {
			break
		}
		results = append(results, item.res)
	}
	return results
}

func (s *ResourceService) UpdateCategory(ctx context.Context, input UpdateResourceCategoryInput) error {
	if strings.TrimSpace(input.ResourceID) == "" || strings.TrimSpace(input.CategoryID) == "" {
		return fmt.Errorf("resource_id and category_id are required")
	}

	category, err := s.categories.GetByID(ctx, input.CategoryID)
	if err != nil {
		return err
	}
	if category == nil {
		return fmt.Errorf("category not found")
	}

	if s.eventsEnabled {
		if err := s.categoryAssignWithEvents(ctx, input.ResourceID, *category); err != nil {
			return err
		}
	} else {
		if err := s.resources.UpdateCategory(ctx, input.ResourceID, input.CategoryID, true); err != nil {
			return err
		}
	}

	if err := s.categories.IncrementAccept(ctx, input.CategoryID); err != nil {
		return err
	}

	return nil
}

// ── event-sourced write helpers (P7 dual-write) ──────────────────────────────

func (s *ResourceService) createWithEvents(ctx context.Context, resource *domain.Resource) error {
	extractedJSON, err := marshalJSON(resource.ExtractedData)
	if err != nil {
		return err
	}

	payload, err := marshalJSON(eventstore.ResourceCreatedPayload{
		URL:               resource.URL,
		Host:              resource.Host,
		Title:             resource.Title,
		Summary:           resource.Summary,
		CategoryID:        resource.CategoryID,
		CategoryName:      resource.CategoryName,
		UserOverride:      resource.UserOverride,
		ExtractedDataJSON: string(extractedJSON),
		CreatedAt:         resource.CreatedAt,
		UpdatedAt:         resource.UpdatedAt,
	})
	if err != nil {
		return err
	}

	evt := eventstore.Event{
		EventID:       uuid.NewString(),
		AggregateID:   resource.ID,
		AggregateType: eventstore.AggregateTypeResource,
		EventType:     eventstore.EventTypeResourceCreated,
		EventVersion:  1,
		Payload:       payload,
	}

	committed, err := s.appendResourceWithTx(ctx, evt)
	if err != nil {
		return err
	}
	s.projectors.ApplyAsync(ctx, committed)
	return nil
}

func (s *ResourceService) updateWithEvents(ctx context.Context, resource *domain.Resource) error {
	payload, err := marshalJSON(eventstore.ResourceUpdatedPayload{
		URL:          resource.URL,
		Host:         resource.Host,
		Title:        resource.Title,
		Summary:      resource.Summary,
		CategoryID:   resource.CategoryID,
		CategoryName: resource.CategoryName,
		UserOverride: resource.UserOverride,
		UpdatedAt:    resource.UpdatedAt,
	})
	if err != nil {
		return err
	}

	version, err := aggregateLatestVersion(ctx, s.eventStore, resource.ID)
	if err != nil {
		return err
	}

	evt := eventstore.Event{
		EventID:       uuid.NewString(),
		AggregateID:   resource.ID,
		AggregateType: eventstore.AggregateTypeResource,
		EventType:     eventstore.EventTypeResourceUpdated,
		EventVersion:  version + 1,
		Payload:       payload,
	}

	committed, err := s.appendResourceWithTx(ctx, evt)
	if err != nil {
		return err
	}
	s.projectors.ApplyAsync(ctx, committed)
	return nil
}

func (s *ResourceService) deleteWithEvents(ctx context.Context, resourceID string) error {
	payload, err := marshalJSON(eventstore.ResourceDeletedPayload{ID: resourceID})
	if err != nil {
		return err
	}

	version, err := aggregateLatestVersion(ctx, s.eventStore, resourceID)
	if err != nil {
		return err
	}

	evt := eventstore.Event{
		EventID:       uuid.NewString(),
		AggregateID:   resourceID,
		AggregateType: eventstore.AggregateTypeResource,
		EventType:     eventstore.EventTypeResourceDeleted,
		EventVersion:  version + 1,
		Payload:       payload,
	}

	committed, err := s.appendResourceWithTx(ctx, evt)
	if err != nil {
		return err
	}
	s.projectors.ApplyAsync(ctx, committed)
	return nil
}

func (s *ResourceService) categoryAssignWithEvents(ctx context.Context, resourceID string, category domain.Category) error {
	payload, err := marshalJSON(eventstore.ResourceCategoryAssignedPayload{
		CategoryID:   category.ID,
		CategoryName: category.Name,
		UserOverride: true,
		UpdatedAt:    time.Now().UTC(),
	})
	if err != nil {
		return err
	}

	version, err := aggregateLatestVersion(ctx, s.eventStore, resourceID)
	if err != nil {
		return err
	}

	evt := eventstore.Event{
		EventID:       uuid.NewString(),
		AggregateID:   resourceID,
		AggregateType: eventstore.AggregateTypeResource,
		EventType:     eventstore.EventTypeResourceCategoryAssigned,
		EventVersion:  version + 1,
		Payload:       payload,
	}

	committed, err := s.appendResourceWithTx(ctx, evt)
	if err != nil {
		return err
	}
	s.projectors.ApplyAsync(ctx, committed)
	return nil
}

// appendResourceWithTx delegates to the package-level OCC retry helper.
func (s *ResourceService) appendResourceWithTx(ctx context.Context, evt eventstore.Event) (eventstore.Event, error) {
	return appendWithTx(ctx, s.eventStore, s.projectors, evt, s.eventObs)
}

func (s *ResourceService) emitCounterIncrementedEvent(ctx context.Context, resourceID string, newCount int) error {
	payload, err := marshalJSON(eventstore.ResourceCounterIncrementedPayload{NewCount: newCount})
	if err != nil {
		return err
	}
	version, err := aggregateLatestVersion(ctx, s.eventStore, resourceID)
	if err != nil {
		return err
	}
	evt := eventstore.Event{
		EventID:       uuid.NewString(),
		AggregateID:   resourceID,
		AggregateType: eventstore.AggregateTypeResource,
		EventType:     eventstore.EventTypeResourceCounterIncremented,
		EventVersion:  version + 1,
		Payload:       payload,
	}
	committed, err := s.appendResourceWithTx(ctx, evt)
	if err != nil {
		return err
	}
	s.projectors.ApplyAsync(ctx, committed)
	return nil
}

func (s *ResourceService) emitArchiveEvent(ctx context.Context, resourceID string, reason domain.ArchiveReason) error {
	payload, err := marshalJSON(eventstore.ResourceArchivedPayload{
		Reason:     string(reason),
		ArchivedAt: time.Now().UTC(),
	})
	if err != nil {
		return err
	}
	version, err := aggregateLatestVersion(ctx, s.eventStore, resourceID)
	if err != nil {
		return err
	}
	evt := eventstore.Event{
		EventID:       uuid.NewString(),
		AggregateID:   resourceID,
		AggregateType: eventstore.AggregateTypeResource,
		EventType:     eventstore.EventTypeResourceArchived,
		EventVersion:  version + 1,
		Payload:       payload,
	}
	committed, err := s.appendResourceWithTx(ctx, evt)
	if err != nil {
		return err
	}
	s.projectors.ApplyAsync(ctx, committed)
	return nil
}

func (s *ResourceService) emitRestoreEvent(ctx context.Context, resourceID string) error {
	payload, err := marshalJSON(eventstore.ResourceRestoredPayload{RestoredAt: time.Now().UTC()})
	if err != nil {
		return err
	}
	version, err := aggregateLatestVersion(ctx, s.eventStore, resourceID)
	if err != nil {
		return err
	}
	evt := eventstore.Event{
		EventID:       uuid.NewString(),
		AggregateID:   resourceID,
		AggregateType: eventstore.AggregateTypeResource,
		EventType:     eventstore.EventTypeResourceRestored,
		EventVersion:  version + 1,
		Payload:       payload,
	}
	committed, err := s.appendResourceWithTx(ctx, evt)
	if err != nil {
		return err
	}
	s.projectors.ApplyAsync(ctx, committed)
	return nil
}

// ── Archive / Restore (WS3) ───────────────────────────────────────────────

func (s *ResourceService) Archive(ctx context.Context, id string, reason domain.ArchiveReason) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("resource id is required")
	}
	if err := s.resources.Archive(ctx, id, reason); err != nil {
		return err
	}
	if s.eventsEnabled {
		_ = s.emitArchiveEvent(ctx, id, reason)
	}
	return nil
}

func (s *ResourceService) Restore(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("resource id is required")
	}
	if err := s.resources.Restore(ctx, id); err != nil {
		return err
	}
	if s.eventsEnabled {
		_ = s.emitRestoreEvent(ctx, id)
	}
	return nil
}

func (s *ResourceService) BulkArchive(ctx context.Context, ids []string, reason domain.ArchiveReason) error {
	if len(ids) == 0 {
		return fmt.Errorf("ids list is empty")
	}
	if len(ids) > 100 {
		return fmt.Errorf("bulk archive limited to 100 resources, got %d", len(ids))
	}
	if err := s.resources.BulkArchive(ctx, ids, reason); err != nil {
		return err
	}
	if s.eventsEnabled {
		for _, id := range ids {
			_ = s.emitArchiveEvent(ctx, id, reason)
		}
	}
	return nil
}

func (s *ResourceService) BulkRestore(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return fmt.Errorf("ids list is empty")
	}
	if len(ids) > 100 {
		return fmt.Errorf("bulk restore limited to 100 resources, got %d", len(ids))
	}
	if err := s.resources.BulkRestore(ctx, ids); err != nil {
		return err
	}
	if s.eventsEnabled {
		for _, id := range ids {
			_ = s.emitRestoreEvent(ctx, id)
		}
	}
	return nil
}

func (s *ResourceService) ListArchived(ctx context.Context, limit, offset int) ([]domain.Resource, error) {
	return s.resources.ListArchived(ctx, limit, offset)
}

func marshalJSON(v any) (json.RawMessage, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal event payload: %w", err)
	}
	return json.RawMessage(b), nil
}

// ── URL and semantic helpers (unchanged) ──────────────────────────────────────

func normalizeURL(rawURL string) (string, string, error) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return "", "", fmt.Errorf("url is required")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", "", fmt.Errorf("invalid url")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", "", fmt.Errorf("url must start with http or https")
	}

	host := strings.TrimPrefix(parsed.Hostname(), "www.")
	return parsed.String(), host, nil
}

func inferTitleFromURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "Untitled Resource"
	}
	host := strings.TrimPrefix(parsed.Hostname(), "www.")
	if host == "" {
		return "Untitled Resource"
	}
	return normalizeCategoryName(strings.ReplaceAll(host, ".", " "))
}

var semanticTokenCleaner = regexp.MustCompile(`[^a-z0-9]+`)

var semanticSynonyms = map[string][]string{
	"ai":           {"artificial", "intelligence", "llm", "ml"},
	"llm":          {"ai", "language", "model"},
	"ml":           {"machine", "learning", "ai"},
	"agent":        {"assistant", "automation"},
	"agents":       {"assistant", "automation", "agent"},
	"graph":        {"network", "node", "relationship"},
	"graphs":       {"network", "node", "relationship", "graph"},
	"knowledge":    {"memory", "information"},
	"productivity": {"workflow", "task", "todo"},
	"todo":         {"task", "checklist"},
	"research":     {"study", "analysis"},
}

func semanticTokenize(input string) []string {
	cleaned := strings.ToLower(semanticTokenCleaner.ReplaceAllString(input, " "))
	parts := strings.Fields(cleaned)
	if len(parts) == 0 {
		return []string{}
	}
	return parts
}

func semanticExpandTokens(tokens []string) map[string]struct{} {
	set := map[string]struct{}{}
	for _, token := range tokens {
		trimmed := strings.TrimSpace(token)
		if trimmed == "" {
			continue
		}
		set[trimmed] = struct{}{}
		if synonyms, ok := semanticSynonyms[trimmed]; ok {
			for _, synonym := range synonyms {
				set[synonym] = struct{}{}
			}
		}
	}
	return set
}

func semanticScore(query string, expandedQueryTokens map[string]struct{}, resource domain.Resource) float64 {
	if len(expandedQueryTokens) == 0 {
		return 0
	}

	resourceText := strings.ToLower(strings.Join([]string{
		resource.Title,
		resource.Summary,
		resource.URL,
		resource.CategoryName,
	}, " "))

	resourceTokens := semanticExpandTokens(semanticTokenize(resourceText))
	if len(resourceTokens) == 0 {
		return 0
	}

	matches := 0
	for token := range expandedQueryTokens {
		if _, ok := resourceTokens[token]; ok {
			matches++
		}
	}

	coverage := float64(matches) / float64(len(expandedQueryTokens))
	score := coverage * 0.85

	trimmedQuery := strings.TrimSpace(strings.ToLower(query))
	if trimmedQuery != "" {
		if strings.Contains(strings.ToLower(resource.Title), trimmedQuery) || strings.Contains(strings.ToLower(resource.Summary), trimmedQuery) {
			score += 0.15
		}
	}

	if score > 1 {
		return 1
	}
	return score
}
