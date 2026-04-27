package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"selfsystems/internal/ai"
	"selfsystems/internal/domain"
)

var (
	ErrDeepProcessingDisabled  = errors.New("deep processing is disabled")
	ErrDeepProcessingQueueFull = errors.New("deep processing queue is full")
)

const deepSummaryMarker = "[deep-processing]"

type DeepProcessingSettings struct {
	Enabled                 bool
	QueueCapacity           int
	WorkerCount             int
	MaxTasksPerMinute       int
	MaxTokensPerDay         int
	ComplexityThreshold     int
	LowCostModel            string
	HighCostModel           string
	LowCostEstimatedTokens  int
	HighCostEstimatedTokens int
}

type DeepTask struct {
	ResourceID string
	EnqueuedAt time.Time
}

type DeepProcessingMetrics struct {
	Enabled                 bool   `json:"enabled"`
	QueueDepth              int    `json:"queue_depth"`
	QueueCapacity           int    `json:"queue_capacity"`
	WorkerCount             int    `json:"worker_count"`
	MaxTasksPerMinute       int    `json:"max_tasks_per_minute"`
	MaxTokensPerDay         int    `json:"max_tokens_per_day"`
	TokensUsedToday         int64  `json:"tokens_used_today"`
	EnqueuedTotal           int64  `json:"enqueued_total"`
	ProcessedTotal          int64  `json:"processed_total"`
	FailedTotal             int64  `json:"failed_total"`
	RejectedTotal           int64  `json:"rejected_total"`
	LowCostRouteTotal       int64  `json:"low_cost_route_total"`
	HighCostRouteTotal      int64  `json:"high_cost_route_total"`
	LastProcessedAt         string `json:"last_processed_at,omitempty"`
	LastError               string `json:"last_error,omitempty"`
	LowCostModel            string `json:"low_cost_model"`
	HighCostModel           string `json:"high_cost_model"`
	LowCostEstimatedTokens  int    `json:"low_cost_estimated_tokens"`
	HighCostEstimatedTokens int    `json:"high_cost_estimated_tokens"`
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

	queue chan DeepTask

	started atomic.Bool

	enqueuedTotal      atomic.Int64
	processedTotal     atomic.Int64
	failedTotal        atomic.Int64
	rejectedTotal      atomic.Int64
	lowCostRouteTotal  atomic.Int64
	highCostRouteTotal atomic.Int64
	tokensUsedToday    atomic.Int64

	lastProcessedAt atomic.Int64

	lastErrorMu sync.RWMutex
	lastError   string

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
	}
	if settings.QueueCapacity > 0 {
		processor.queue = make(chan DeepTask, settings.QueueCapacity)
	}
	return processor
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

	select {
	case p.queue <- task:
		p.enqueuedTotal.Add(1)
		return nil
	default:
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
	lastError := p.lastError
	p.lastErrorMu.RUnlock()

	return DeepProcessingMetrics{
		Enabled:                 p.settings.Enabled,
		QueueDepth:              queueDepth,
		QueueCapacity:           p.settings.QueueCapacity,
		WorkerCount:             p.settings.WorkerCount,
		MaxTasksPerMinute:       p.settings.MaxTasksPerMinute,
		MaxTokensPerDay:         p.settings.MaxTokensPerDay,
		TokensUsedToday:         p.tokensUsedToday.Load(),
		EnqueuedTotal:           p.enqueuedTotal.Load(),
		ProcessedTotal:          p.processedTotal.Load(),
		FailedTotal:             p.failedTotal.Load(),
		RejectedTotal:           p.rejectedTotal.Load(),
		LowCostRouteTotal:       p.lowCostRouteTotal.Load(),
		HighCostRouteTotal:      p.highCostRouteTotal.Load(),
		LastProcessedAt:         lastProcessedAt,
		LastError:               lastError,
		LowCostModel:            p.settings.LowCostModel,
		HighCostModel:           p.settings.HighCostModel,
		LowCostEstimatedTokens:  p.settings.LowCostEstimatedTokens,
		HighCostEstimatedTokens: p.settings.HighCostEstimatedTokens,
	}
}

func (p *DeepProcessor) runWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case task := <-p.queue:
			if err := p.processTask(ctx, task); err != nil {
				p.failedTotal.Add(1)
				p.setLastError(err.Error())
			}
		}
	}
}

func (p *DeepProcessor) processTask(ctx context.Context, task DeepTask) error {
	if err := p.acquireThroughputSlot(ctx); err != nil {
		return err
	}

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

	route, model, estimatedTokens, complexity := p.selectRoute(*resource)
	if !p.reserveTokenBudget(estimatedTokens) {
		p.rejectedTotal.Add(1)
		return fmt.Errorf("deep processing token budget exceeded")
	}

	deepSummary := p.buildDeepSummary(resource.Summary, route, model, complexity, estimatedTokens)
	categoryName := p.suggestCategoryName(ctx, *resource, route)

	updateInput := UpdateResourceInput{
		ID:      resource.ID,
		URL:     resource.URL,
		Title:   resource.Title,
		Summary: deepSummary,
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
	p.lastErrorMu.Lock()
	p.lastError = strings.TrimSpace(message)
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

	urlLower := strings.ToLower(resource.URL)
	if strings.Contains(urlLower, "github") || strings.Contains(urlLower, "docs") || strings.Contains(urlLower, "research") {
		score += 2
	}
	if strings.Contains(urlLower, "?") {
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
