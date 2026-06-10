package service

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"selfsystems/internal/domain"
)

// ArchiveWorkerConfig controls which auto-archive triggers are active.
type ArchiveWorkerConfig struct {
	// DeadLinksEnabled enables HTTP HEAD checks; resources returning 404 or
	// connection refused are auto-archived with reason "dead_link".
	DeadLinksEnabled bool
	// ExpiredEventsEnabled archives resources whose extracted event_date is in
	// the past with reason "expired".
	ExpiredEventsEnabled bool
	// HTTPTimeout is the per-request timeout for dead link checks. Default 10s.
	HTTPTimeout time.Duration
}

// ArchiveWorker is a background job that runs once per day and auto-archives
// stale resources based on configurable triggers (WS4).
type ArchiveWorker struct {
	resources domain.ResourceRepository
	svc       *ResourceService
	cfg       ArchiveWorkerConfig
	client    *http.Client
}

// NewArchiveWorker creates the worker. cfg.HTTPTimeout defaults to 10s when zero.
func NewArchiveWorker(resources domain.ResourceRepository, svc *ResourceService, cfg ArchiveWorkerConfig) *ArchiveWorker {
	if cfg.HTTPTimeout <= 0 {
		cfg.HTTPTimeout = 10 * time.Second
	}
	return &ArchiveWorker{
		resources: resources,
		svc:       svc,
		cfg:       cfg,
		client: &http.Client{
			Timeout: cfg.HTTPTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
	}
}

// Run executes one full archive cycle synchronously and returns the count of
// resources archived. Intended to be called from a daily ticker goroutine.
func (w *ArchiveWorker) Run(ctx context.Context) (int, error) {
	archived := 0

	// Page through all non-archived resources in batches of 200.
	offset := 0
	const batchSize = 200
	for {
		resources, err := w.resources.List(ctx, batchSize, offset)
		if err != nil {
			return archived, err
		}
		if len(resources) == 0 {
			break
		}

		for _, r := range resources {
			if w.cfg.DeadLinksEnabled {
				if w.isDeadLink(ctx, r.URL) {
					if err := w.svc.Archive(ctx, r.ID, domain.ArchiveReasonDeadLink); err != nil {
						slog.Warn("archive_worker: failed to archive dead link",
							"resource_id", r.ID, "url", r.URL, "error", err)
					} else {
						slog.Info("archive_worker: archived dead link",
							"resource_id", r.ID, "url", r.URL)
						archived++
						continue
					}
				}
			}

			if w.cfg.ExpiredEventsEnabled {
				if w.isExpiredEvent(r) {
					if err := w.svc.Archive(ctx, r.ID, domain.ArchiveReasonExpired); err != nil {
						slog.Warn("archive_worker: failed to archive expired event",
							"resource_id", r.ID, "url", r.URL, "error", err)
					} else {
						slog.Info("archive_worker: archived expired event",
							"resource_id", r.ID, "url", r.URL)
						archived++
					}
				}
			}
		}

		if len(resources) < batchSize {
			break
		}
		offset += batchSize
	}

	slog.Info("archive_worker: cycle complete", "archived", archived)
	return archived, nil
}

// Start launches Run on a daily ticker. Call cancel to stop the goroutine.
func (w *ArchiveWorker) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		// Run once immediately on startup.
		if _, err := w.Run(ctx); err != nil {
			slog.Warn("archive_worker: initial run failed", "error", err)
		}
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := w.Run(ctx); err != nil {
					slog.Warn("archive_worker: daily run failed", "error", err)
				}
			}
		}
	}()
}

func (w *ArchiveWorker) isDeadLink(ctx context.Context, url string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return false
	}
	resp, err := w.client.Do(req)
	if err != nil {
		return true // connection refused or DNS failure → treat as dead
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusNotFound || resp.StatusCode >= 500
}

func (w *ArchiveWorker) isExpiredEvent(r domain.Resource) bool {
	eventDate := r.ExtractedData.EventDate
	if eventDate == "" {
		return false
	}
	t, err := time.Parse(time.RFC3339, eventDate)
	if err != nil {
		return false
	}
	return t.Before(time.Now().UTC())
}
