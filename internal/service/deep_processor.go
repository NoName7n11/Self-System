package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"selfsystems/internal/ai"
	"selfsystems/internal/domain"
	"selfsystems/internal/extractor"
)

var (
	ErrDeepProcessingDisabled  = errors.New("deep processing is disabled")
	ErrDeepProcessingQueueFull = errors.New("deep processing queue is full")
)

const deepSummaryMarker = "[deep-processing]"

const deepRecentErrorsCap = 20

type DeepProcessingSettings struct {
	Enabled                     bool
	QueueCapacity               int
	WorkerCount                 int
	BatchSize                   int
	MaxTasksPerMinute           int
	MaxTokensPerDay             int
	MinReprocessIntervalSeconds int
	ComplexityThreshold         int
	LowCostModel                string
	HighCostModel               string
	LowCostEstimatedTokens      int
	HighCostEstimatedTokens     int
	BudgetStatePath             string
}

type DeepTask struct {
	ResourceID string
	EnqueuedAt time.Time
}

type DeepProcessingMetrics struct {
	Enabled                     bool     `json:"enabled"`
	QueueDepth                  int      `json:"queue_depth"`
	QueueCapacity               int      `json:"queue_capacity"`
	WorkerCount                 int      `json:"worker_count"`
	BatchSize                   int      `json:"batch_size"`
	MaxTasksPerMinute           int      `json:"max_tasks_per_minute"`
	MaxTokensPerDay             int      `json:"max_tokens_per_day"`
	MinReprocessIntervalSeconds int      `json:"min_reprocess_interval_seconds"`
	TokensUsedToday             int64    `json:"tokens_used_today"`
	EnqueuedTotal               int64    `json:"enqueued_total"`
	ProcessedTotal              int64    `json:"processed_total"`
	FailedTotal                 int64    `json:"failed_total"`
	RejectedTotal               int64    `json:"rejected_total"`
	DedupedTotal                int64    `json:"deduped_total"`
	SkippedTotal                int64    `json:"skipped_total"`
	LowCostRouteTotal           int64    `json:"low_cost_route_total"`
	HighCostRouteTotal          int64    `json:"high_cost_route_total"`
	LastProcessedAt             string   `json:"last_processed_at,omitempty"`
	LastError                   string   `json:"last_error,omitempty"`
	RecentErrors                []string `json:"recent_errors,omitempty"`
	LowCostModel                string   `json:"low_cost_model"`
	HighCostModel               string   `json:"high_cost_model"`
	LowCostEstimatedTokens      int      `json:"low_cost_estimated_tokens"`
	HighCostEstimatedTokens     int      `json:"high_cost_estimated_tokens"`
}

type DeepProcessingHealth struct {
	Status        string                `json:"status"`
	DeepEnabled   bool                  `json:"deep_enabled"`
	Backlog       int                   `json:"backlog"`
	QueueCapacity int                   `json:"queue_capacity"`
	Metrics       DeepProcessingMetrics `json:"metrics"`
}

type DeepProcessor struct {
	resources   *ResourceService
	categories  domain.CategoryRepository
	categorySvc *CategoryService
	manager     *ai.Manager
	settings    DeepProcessingSettings

	// Extraction workers (Change 6 WS5). All are optional; nil = feature disabled.
	contentFetcher *extractor.ContentFetcher
	pdfExtractor   *extractor.PDFExtractor
	imageExtractor *extractor.ImageExtractor
	eventDetector  *extractor.EventDetector
	reminderSvc    *ReminderService

	// Embedding generation (Change 7 WS2). Optional; nil = disabled.
	embeddingSvc *EmbeddingService

	queue chan DeepTask

	pendingMu sync.Mutex
	pending   map[string]struct{}

	started atomic.Bool

	enqueuedTotal      atomic.Int64
	processedTotal     atomic.Int64
	failedTotal        atomic.Int64
	rejectedTotal      atomic.Int64
	dedupedTotal       atomic.Int64
	skippedTotal       atomic.Int64
	lowCostRouteTotal  atomic.Int64
	highCostRouteTotal atomic.Int64
	tokensUsedToday    atomic.Int64

	lastProcessedAt atomic.Int64

	lastErrorMu  sync.RWMutex
	recentErrors []string

	throughputMu      sync.Mutex
	windowStart       time.Time
	processedInWindow int

	budgetMu   sync.Mutex
	budgetDate string
	budgetUsed int64
}

func NewDeepProcessor(
	resources *ResourceService,
	categories domain.CategoryRepository,
	categorySvc *CategoryService,
	manager *ai.Manager,
	settings DeepProcessingSettings,
) *DeepProcessor {
	applyDeepSettingsDefaults(&settings)

	processor := &DeepProcessor{
		resources:   resources,
		categories:  categories,
		categorySvc: categorySvc,
		manager:     manager,
		settings:    settings,
		pending:     make(map[string]struct{}),
	}
	if settings.QueueCapacity > 0 {
		processor.queue = make(chan DeepTask, settings.QueueCapacity)
	}
	processor.loadBudgetState()
	return processor
}

// WithContentFetcher enables raw URL fetching for the PDF and image extraction paths.
func (p *DeepProcessor) WithContentFetcher(f *extractor.ContentFetcher) *DeepProcessor {
	p.contentFetcher = f
	return p
}

// WithPDFExtractor enables PDF text extraction for resources whose URL ends in .pdf.
func (p *DeepProcessor) WithPDFExtractor(e *extractor.PDFExtractor) *DeepProcessor {
	p.pdfExtractor = e
	return p
}

// WithImageExtractor enables image classification and thumbnail generation for
// resources whose URL ends in a common image extension.
func (p *DeepProcessor) WithImageExtractor(e *extractor.ImageExtractor) *DeepProcessor {
	p.imageExtractor = e
	return p
}

// WithEventDetector enables event signal detection and auto-reminder creation.
func (p *DeepProcessor) WithEventDetector(e *extractor.EventDetector) *DeepProcessor {
	p.eventDetector = e
	return p
}

// WithReminderService wires the reminder service so that detected future events
// automatically generate linked reminders.
func (p *DeepProcessor) WithReminderService(svc *ReminderService) *DeepProcessor {
	p.reminderSvc = svc
	return p
}

// WithEmbeddingService enables embedding generation during deep processing.
func (p *DeepProcessor) WithEmbeddingService(svc *EmbeddingService) *DeepProcessor {
	p.embeddingSvc = svc
	return p
}

// ProcessDirect runs the full deep-processing pipeline synchronously for a
// single resource, bypassing the queue. Intended for tests and manual tooling.
func (p *DeepProcessor) ProcessDirect(ctx context.Context, resourceID string) error {
	return p.processTask(ctx, DeepTask{
		ResourceID: resourceID,
		EnqueuedAt: time.Now().UTC(),
	})
}

func (p *DeepProcessor) Start(ctx context.Context) {
	if p == nil || !p.settings.Enabled || p.queue == nil {
		return
	}
	if !p.started.CompareAndSwap(false, true) {
		return
	}

	for worker := 0; worker < p.settings.WorkerCount; worker++ {
		go p.runWorker(ctx)
	}
}

func (p *DeepProcessor) Enqueue(task DeepTask) error {
	if p == nil || !p.settings.Enabled {
		return ErrDeepProcessingDisabled
	}
	if p.queue == nil {
		return ErrDeepProcessingDisabled
	}

	resourceID := strings.TrimSpace(task.ResourceID)
	if resourceID == "" {
		return fmt.Errorf("resource id is required")
	}
	task.ResourceID = resourceID
	if task.EnqueuedAt.IsZero() {
		task.EnqueuedAt = time.Now().UTC()
	}

	if p.markPending(resourceID) {
		p.dedupedTotal.Add(1)
		return nil
	}

	select {
	case p.queue <- task:
		p.enqueuedTotal.Add(1)
		return nil
	default:
		p.clearPending(resourceID)
		p.rejectedTotal.Add(1)
		p.setLastError("deep processing queue is full")
		return ErrDeepProcessingQueueFull
	}
}

func (p *DeepProcessor) Reprocess(ctx context.Context, resourceID string) error {
	_ = ctx
	return p.Enqueue(DeepTask{ResourceID: resourceID})
}

func (p *DeepProcessor) Health() DeepProcessingHealth {
	metrics := p.Metrics()
	status := "ok"
	if !metrics.Enabled {
		status = "disabled"
	}

	return DeepProcessingHealth{
		Status:        status,
		DeepEnabled:   metrics.Enabled,
		Backlog:       metrics.QueueDepth,
		QueueCapacity: metrics.QueueCapacity,
		Metrics:       metrics,
	}
}

func (p *DeepProcessor) Metrics() DeepProcessingMetrics {
	if p == nil {
		return DeepProcessingMetrics{}
	}

	queueDepth := 0
	if p.queue != nil {
		queueDepth = len(p.queue)
	}

	lastProcessedAt := ""
	if unix := p.lastProcessedAt.Load(); unix > 0 {
		lastProcessedAt = time.Unix(unix, 0).UTC().Format(time.RFC3339)
	}

	p.lastErrorMu.RLock()
	recent := append([]string(nil), p.recentErrors...)
	p.lastErrorMu.RUnlock()
	lastError := ""
	if len(recent) > 0 {
		lastError = recent[len(recent)-1]
	}

	return DeepProcessingMetrics{
		Enabled:                     p.settings.Enabled,
		QueueDepth:                  queueDepth,
		QueueCapacity:               p.settings.QueueCapacity,
		WorkerCount:                 p.settings.WorkerCount,
		BatchSize:                   p.settings.BatchSize,
		MaxTasksPerMinute:           p.settings.MaxTasksPerMinute,
		MaxTokensPerDay:             p.settings.MaxTokensPerDay,
		MinReprocessIntervalSeconds: p.settings.MinReprocessIntervalSeconds,
		TokensUsedToday:             p.tokensUsedToday.Load(),
		EnqueuedTotal:               p.enqueuedTotal.Load(),
		ProcessedTotal:              p.processedTotal.Load(),
		FailedTotal:                 p.failedTotal.Load(),
		RejectedTotal:               p.rejectedTotal.Load(),
		DedupedTotal:                p.dedupedTotal.Load(),
		SkippedTotal:                p.skippedTotal.Load(),
		LowCostRouteTotal:           p.lowCostRouteTotal.Load(),
		HighCostRouteTotal:          p.highCostRouteTotal.Load(),
		LastProcessedAt:             lastProcessedAt,
		LastError:                   lastError,
		RecentErrors:                recent,
		LowCostModel:                p.settings.LowCostModel,
		HighCostModel:               p.settings.HighCostModel,
		LowCostEstimatedTokens:      p.settings.LowCostEstimatedTokens,
		HighCostEstimatedTokens:     p.settings.HighCostEstimatedTokens,
	}
}

func (p *DeepProcessor) runWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case task := <-p.queue:
			batch := p.collectBatch(task)
			for _, item := range batch {
				if err := p.processTask(ctx, item); err != nil {
					p.failedTotal.Add(1)
					p.setLastError(err.Error())
				}
				p.clearPending(item.ResourceID)
			}
		}
	}
}

func (p *DeepProcessor) processTask(ctx context.Context, task DeepTask) error {
	if p.resources == nil {
		return fmt.Errorf("resource service is not configured for deep processing")
	}

	resource, err := p.resources.GetByID(ctx, task.ResourceID)
	if err != nil {
		return err
	}
	if resource == nil {
		return fmt.Errorf("resource %q not found for deep processing", task.ResourceID)
	}
	if p.shouldSkipResource(*resource) {
		p.skippedTotal.Add(1)
		return nil
	}

	// Content extraction (PDF / image) — runs before the token budget is reserved
	// so extraction always happens regardless of AI quota state.
	p.runExtractionForResource(ctx, resource)

	// Event detection + auto-reminder creation — also pre-budget.
	p.runEventDetection(ctx, resource)

	// Embedding generation — token cost is tracked against the daily budget.
	p.runEmbedding(ctx, resource)

	if err := p.acquireThroughputSlot(ctx); err != nil {
		return err
	}

	route, model, estimatedTokens, complexity := p.selectRoute(*resource)
	if !p.reserveTokenBudget(estimatedTokens) {
		p.rejectedTotal.Add(1)
		return fmt.Errorf("deep processing token budget exceeded")
	}

	// Attempt real AI enrichment; fall back to the annotation stub when the
	// manager has no enrichment provider or the budget is exhausted.
	summary, enriched := p.runEnrichment(ctx, resource)
	if !enriched {
		summary = p.buildDeepSummary(resource.Summary, route, model, complexity, estimatedTokens)
	}

	categoryName := p.suggestCategoryName(ctx, *resource, route)

	updateInput := UpdateResourceInput{
		ID:      resource.ID,
		URL:     resource.URL,
		Title:   resource.Title,
		Summary: summary,
	}
	if categoryName != "" && !strings.EqualFold(strings.TrimSpace(resource.CategoryName), categoryName) {
		updateInput.CategoryName = categoryName
	}

	if _, err := p.resources.Update(ctx, updateInput); err != nil {
		return err
	}

	p.processedTotal.Add(1)
	if route == "high" {
		p.highCostRouteTotal.Add(1)
	} else {
		p.lowCostRouteTotal.Add(1)
	}
	p.lastProcessedAt.Store(time.Now().UTC().Unix())

	return nil
}

// ---- Change 6 WS5: extraction and event-detection helpers ------------------

// runExtractionForResource dispatches PDF or image extraction for resources
// whose URL extension matches. URL/HTML resources were already handled by the
// skim extractor in WS1; nothing to do for them here.
func (p *DeepProcessor) runExtractionForResource(ctx context.Context, resource *domain.Resource) {
	if p.contentFetcher == nil || resource.URL == "" {
		return
	}
	switch inferSourceType(resource.URL) {
	case "pdf":
		if p.pdfExtractor != nil {
			p.runPDFExtraction(ctx, resource)
		}
	case "image":
		if p.imageExtractor != nil {
			p.runImageExtraction(ctx, resource)
		}
	}
}

func (p *DeepProcessor) runPDFExtraction(ctx context.Context, resource *domain.Resource) {
	fetchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	fetched, err := p.contentFetcher.Fetch(fetchCtx, resource.URL)
	if err != nil {
		return
	}

	result, err := p.pdfExtractor.Extract(ctx, fetched.Content)
	if err != nil {
		return
	}

	data := resource.ExtractedData
	data.PDFPageCount = result.PageCount
	data.PDFText = result.Text
	if data.MainText == "" {
		data.MainText = result.Text
	}

	updateCtx, updateCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer updateCancel()
	if err := p.resources.UpdateExtractedData(updateCtx, resource.ID, data); err != nil {
		slog.Warn("deep processor: failed to write PDF extracted_data", "resource_id", resource.ID, "error", err)
		p.setLastError(err.Error())
	}
	resource.ExtractedData = data
}

func (p *DeepProcessor) runImageExtraction(ctx context.Context, resource *domain.Resource) {
	fetchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	fetched, err := p.contentFetcher.Fetch(fetchCtx, resource.URL)
	if err != nil {
		return
	}

	parsedURL, _ := url.Parse(resource.URL)
	filename := ""
	if parsedURL != nil {
		filename = filepath.Base(parsedURL.Path)
	}

	result, err := p.imageExtractor.Extract(ctx, fetched.Content, filename)
	if err != nil {
		return
	}

	data := resource.ExtractedData
	data.ImageType = result.ImageType
	data.ImageFormat = result.Format
	data.ImageWidth = result.Width
	data.ImageHeight = result.Height
	data.ThumbnailBase64 = result.ThumbnailBase64()

	updateCtx, updateCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer updateCancel()
	if err := p.resources.UpdateExtractedData(updateCtx, resource.ID, data); err != nil {
		slog.Warn("deep processor: failed to write image extracted_data", "resource_id", resource.ID, "error", err)
		p.setLastError(err.Error())
	}
	resource.ExtractedData = data
}

// runEventDetection scans the resource's extracted text for event signals and
// auto-creates reminders for signals that carry a future date.
func (p *DeepProcessor) runEventDetection(ctx context.Context, resource *domain.Resource) {
	if p.eventDetector == nil {
		return
	}

	text := resource.ExtractedData.MainText
	if text == "" {
		text = resource.ExtractedData.ExtractedDescription
	}
	if text == "" {
		text = resource.Title + " " + resource.Summary
	}
	if strings.TrimSpace(text) == "" {
		return
	}

	result := p.eventDetector.Detect(ctx, text)
	if !result.IsEvent || p.reminderSvc == nil {
		return
	}

	for _, signal := range result.Signals {
		if !signal.HasFutureDate() {
			continue
		}
		// Dedup: skip if a reminder already exists for this resource on the same day.
		if p.reminderExistsForDate(ctx, resource.ID, signal.Date) {
			continue
		}
		title := "Event: " + resource.Title
		if len([]rune(title)) > 200 {
			title = string([]rune(title)[:200])
		}
		if _, err := p.reminderSvc.Create(ctx, CreateReminderInput{
			Title:      title,
			Message:    signal.Snippet,
			RemindAt:   signal.Date,
			ResourceID: &resource.ID,
		}); err != nil {
			slog.Warn("deep processor: failed to create event reminder",
				"resource_id", resource.ID, "keyword", signal.Keyword, "error", err)
			p.setLastError(err.Error())
		}
	}
}

// reminderExistsForDate returns true when a reminder already exists for the
// given resource on the same UTC calendar day as date. Prevents duplicate
// reminders when a resource is reprocessed (Finding 6).
func (p *DeepProcessor) reminderExistsForDate(ctx context.Context, resourceID string, date time.Time) bool {
	if p.reminderSvc == nil {
		return false
	}
	existing, err := p.reminderSvc.List(ctx, 200, 0)
	if err != nil {
		return false
	}
	targetDay := date.UTC().Truncate(24 * time.Hour)
	for _, r := range existing {
		if r.ResourceID == nil || *r.ResourceID != resourceID {
			continue
		}
		if r.RemindAt.UTC().Truncate(24 * time.Hour).Equal(targetDay) {
			return true
		}
	}
	return false
}

// runEnrichment calls the AI manager's EnrichResource to generate a real summary,
// key points, and entities. Returns the summary and true on success; returns
// empty string and false when the manager has no enrichment provider or the call
// fails, so the caller can fall back to the annotation stub.
// Key points and entities are persisted into extracted_data.
func (p *DeepProcessor) runEnrichment(ctx context.Context, resource *domain.Resource) (summary string, ok bool) {
	if p.manager == nil {
		return "", false
	}

	content := embeddingTextForResource(*resource)
	if strings.TrimSpace(content) == "" {
		return "", false
	}

	result, err := p.manager.EnrichResource(ctx, ai.EnrichmentInput{
		Title:   resource.Title,
		URL:     resource.URL,
		Content: content,
	})
	if err != nil {
		return "", false
	}
	if strings.TrimSpace(result.Summary) == "" {
		return "", false
	}

	// Persist key points and entities back to extracted_data.
	data := resource.ExtractedData
	if len(result.KeyPoints) > 0 {
		data.KeyPoints = result.KeyPoints
	}
	if len(result.Entities) > 0 {
		data.Entities = result.Entities
	}
	updateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = p.resources.UpdateExtractedData(updateCtx, resource.ID, data)
	resource.ExtractedData = data

	return result.Summary, true
}

// runEmbedding generates and stores an embedding for the resource's content.
// The estimated token cost is reserved against the daily budget; if the budget
// is exhausted the embedding is skipped (non-fatal).
func (p *DeepProcessor) runEmbedding(ctx context.Context, resource *domain.Resource) {
	if p.embeddingSvc == nil {
		return
	}
	text := embeddingTextForResource(*resource)
	if strings.TrimSpace(text) == "" {
		return
	}

	if !p.reserveTokenBudget(estimateEmbeddingTokens(text)) {
		return // budget exhausted — skip embedding this cycle
	}

	embedCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	if _, err := p.embeddingSvc.EmbedResource(embedCtx, resource.ID, text); err != nil {
		slog.Warn("deep processor: failed to embed resource", "resource_id", resource.ID, "error", err)
		p.setLastError(err.Error())
	}
}

// embeddingTextForResource assembles the best available text for embedding:
// title + summary + extracted body/PDF/OCR text.
func embeddingTextForResource(resource domain.Resource) string {
	parts := []string{resource.Title, resource.Summary}
	ed := resource.ExtractedData
	if ed.MainText != "" {
		parts = append(parts, ed.MainText)
	}
	if ed.PDFText != "" {
		parts = append(parts, ed.PDFText)
	}
	if ed.OCRText != "" {
		parts = append(parts, ed.OCRText)
	}
	if ed.ExtractedDescription != "" {
		parts = append(parts, ed.ExtractedDescription)
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

// inferSourceType returns "pdf", "image", or "url" based on the URL path extension.
func inferSourceType(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "url"
	}
	path := strings.ToLower(u.Path)
	if strings.HasSuffix(path, ".pdf") {
		return "pdf"
	}
	for _, ext := range []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp"} {
		if strings.HasSuffix(path, ext) {
			return "image"
		}
	}
	return "url"
}

func (p *DeepProcessor) collectBatch(first DeepTask) []DeepTask {
	batchSize := p.settings.BatchSize
	if batchSize <= 1 || p.queue == nil {
		return []DeepTask{first}
	}

	items := make([]DeepTask, 0, batchSize)
	items = append(items, first)
	for len(items) < batchSize {
		select {
		case next := <-p.queue:
			items = append(items, next)
		default:
			return items
		}
	}

	return items
}

func (p *DeepProcessor) shouldSkipResource(resource domain.Resource) bool {
	minInterval := time.Duration(p.settings.MinReprocessIntervalSeconds) * time.Second
	if minInterval <= 0 {
		return false
	}

	processedAt, ok := parseDeepProcessedAt(resource.Summary)
	if !ok {
		return false
	}
	if !resource.UpdatedAt.IsZero() && resource.UpdatedAt.After(processedAt) {
		return false
	}

	now := time.Now().UTC()
	if processedAt.After(now) {
		return false
	}

	return now.Sub(processedAt) < minInterval
}

func parseDeepProcessedAt(summary string) (time.Time, bool) {
	markerIndex := strings.Index(summary, deepSummaryMarker)
	if markerIndex == -1 {
		return time.Time{}, false
	}

	segment := summary[markerIndex:]
	key := "processed_at="
	keyIndex := strings.Index(segment, key)
	if keyIndex == -1 {
		return time.Time{}, false
	}

	start := keyIndex + len(key)
	value := segment[start:]
	if cut := strings.IndexAny(value, " \n\t\r"); cut != -1 {
		value = value[:cut]
	}

	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, false
	}

	return parsed.UTC(), true
}

func (p *DeepProcessor) markPending(resourceID string) bool {
	if p == nil {
		return false
	}

	resourceID = strings.TrimSpace(resourceID)
	if resourceID == "" {
		return false
	}

	p.pendingMu.Lock()
	defer p.pendingMu.Unlock()

	if p.pending == nil {
		p.pending = make(map[string]struct{})
	}
	if _, exists := p.pending[resourceID]; exists {
		return true
	}

	p.pending[resourceID] = struct{}{}
	return false
}

func (p *DeepProcessor) clearPending(resourceID string) {
	if p == nil {
		return
	}

	resourceID = strings.TrimSpace(resourceID)
	if resourceID == "" {
		return
	}

	p.pendingMu.Lock()
	if p.pending != nil {
		delete(p.pending, resourceID)
	}
	p.pendingMu.Unlock()
}

func (p *DeepProcessor) suggestCategoryName(ctx context.Context, resource domain.Resource, route string) string {
	if p.manager == nil || p.categories == nil || route != "high" {
		return ""
	}

	items, err := p.categories.List(ctx)
	if err != nil {
		return ""
	}

	names := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.Name) != "" {
			names = append(names, item.Name)
		}
	}

	output, err := p.manager.ClassifySkim(ctx, ai.ClassificationInput{
		URL:                resource.URL,
		Title:              resource.Title,
		Summary:            resource.Summary,
		ExistingCategories: names,
	})
	if err != nil {
		return ""
	}
	if output.Confidence < 0.82 {
		return ""
	}

	return normalizeCategoryName(output.SuggestedCategory)
}

func (p *DeepProcessor) selectRoute(resource domain.Resource) (route, model string, estimatedTokens, complexity int) {
	complexity = deepComplexityScore(resource)
	if complexity >= p.settings.ComplexityThreshold {
		return "high", p.settings.HighCostModel, p.settings.HighCostEstimatedTokens, complexity
	}
	return "low", p.settings.LowCostModel, p.settings.LowCostEstimatedTokens, complexity
}

func (p *DeepProcessor) acquireThroughputSlot(ctx context.Context) error {
	if p.settings.MaxTasksPerMinute <= 0 {
		return nil
	}

	for {
		p.throughputMu.Lock()
		now := time.Now().UTC()
		if p.windowStart.IsZero() || now.Sub(p.windowStart) >= time.Minute {
			p.windowStart = now
			p.processedInWindow = 0
		}

		if p.processedInWindow < p.settings.MaxTasksPerMinute {
			p.processedInWindow++
			p.throughputMu.Unlock()
			return nil
		}

		waitDuration := time.Until(p.windowStart.Add(time.Minute))
		if waitDuration < 150*time.Millisecond {
			waitDuration = 150 * time.Millisecond
		}
		p.throughputMu.Unlock()

		timer := time.NewTimer(waitDuration)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (p *DeepProcessor) reserveTokenBudget(tokens int) bool {
	if p.settings.MaxTokensPerDay <= 0 {
		return true
	}
	if tokens <= 0 {
		tokens = 1
	}

	p.budgetMu.Lock()
	defer p.budgetMu.Unlock()

	today := time.Now().UTC().Format("2006-01-02")
	if p.budgetDate != today {
		p.budgetDate = today
		p.budgetUsed = 0
	}

	if p.budgetUsed+int64(tokens) > int64(p.settings.MaxTokensPerDay) {
		return false
	}

	p.budgetUsed += int64(tokens)
	p.tokensUsedToday.Store(p.budgetUsed)
	p.persistBudgetState()
	return true
}

func (p *DeepProcessor) buildDeepSummary(existingSummary, route, model string, complexity, estimatedTokens int) string {
	line := fmt.Sprintf("%s route=%s model=%s complexity=%d estimated_tokens=%d processed_at=%s", deepSummaryMarker, route, model, complexity, estimatedTokens, time.Now().UTC().Format(time.RFC3339))

	trimmed := strings.TrimSpace(existingSummary)
	if trimmed == "" {
		return line
	}

	lines := strings.Split(trimmed, "\n")
	filtered := make([]string, 0, len(lines))
	for _, entry := range lines {
		if strings.Contains(entry, deepSummaryMarker) {
			continue
		}
		filtered = append(filtered, entry)
	}

	base := strings.TrimSpace(strings.Join(filtered, "\n"))
	if base == "" {
		return line
	}
	return base + "\n\n" + line
}

func (p *DeepProcessor) setLastError(message string) {
	msg := strings.TrimSpace(message)
	if msg == "" {
		return
	}
	p.lastErrorMu.Lock()
	p.recentErrors = append(p.recentErrors, msg)
	if len(p.recentErrors) > deepRecentErrorsCap {
		p.recentErrors = p.recentErrors[len(p.recentErrors)-deepRecentErrorsCap:]
	}
	p.lastErrorMu.Unlock()
}

func deepComplexityScore(resource domain.Resource) int {
	score := 0
	wordCount := len(strings.Fields(resource.Title + " " + resource.Summary))

	if wordCount > 40 {
		score += 3
	} else if wordCount > 20 {
		score += 2
	} else {
		score++
	}

	if len(strings.TrimSpace(resource.Summary)) > 220 {
		score += 2
	}

	hostLower := deepHostLower(resource.URL)
	if strings.Contains(hostLower, "github") || strings.Contains(hostLower, "docs") || strings.Contains(hostLower, "research") {
		score += 2
	}
	if strings.Contains(resource.URL, "?") {
		score++
	}

	return score
}

func applyDeepSettingsDefaults(settings *DeepProcessingSettings) {
	if settings.QueueCapacity <= 0 {
		settings.QueueCapacity = 256
	}
	if settings.WorkerCount <= 0 {
		settings.WorkerCount = 1
	}
	if settings.BatchSize <= 0 {
		settings.BatchSize = 8
	}
	if settings.MaxTasksPerMinute <= 0 {
		settings.MaxTasksPerMinute = 30
	}
	if settings.MaxTokensPerDay <= 0 {
		settings.MaxTokensPerDay = 200000
	}
	if settings.ComplexityThreshold <= 0 {
		settings.ComplexityThreshold = 6
	}
	if strings.TrimSpace(settings.LowCostModel) == "" {
		settings.LowCostModel = "gpt-4o-mini"
	}
	if strings.TrimSpace(settings.HighCostModel) == "" {
		settings.HighCostModel = "gpt-4o"
	}
	if settings.LowCostEstimatedTokens <= 0 {
		settings.LowCostEstimatedTokens = 250
	}
	if settings.HighCostEstimatedTokens <= 0 {
		settings.HighCostEstimatedTokens = 1200
	}
}

func deepHostLower(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(parsed.Hostname()))
}

type deepBudgetState struct {
	Date string `json:"date"`
	Used int64  `json:"used"`
}

func (p *DeepProcessor) loadBudgetState() {
	path := strings.TrimSpace(p.settings.BudgetStatePath)
	if path == "" {
		return
	}

	bytes, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var state deepBudgetState
	if err := json.Unmarshal(bytes, &state); err != nil {
		return
	}

	p.budgetMu.Lock()
	p.budgetDate = strings.TrimSpace(state.Date)
	p.budgetUsed = state.Used
	p.budgetMu.Unlock()
	p.tokensUsedToday.Store(state.Used)
}

func (p *DeepProcessor) persistBudgetState() {
	path := strings.TrimSpace(p.settings.BudgetStatePath)
	if path == "" {
		return
	}

	state := deepBudgetState{
		Date: p.budgetDate,
		Used: p.budgetUsed,
	}
	payload, err := json.Marshal(state)
	if err != nil {
		return
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	_ = os.WriteFile(path, payload, 0o600)
}
