package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"

	"selfsystems/internal/domain"
	"selfsystems/internal/eventstore"
)

type CategoryService struct {
	repo          domain.CategoryRepository
	evtStore      eventstore.Store
	projectors    *eventstore.ProjectorRegistry
	eventsEnabled bool
	eventObs      *eventstore.EventObservability
}

// CategoryServiceOption configures a CategoryService.
type CategoryServiceOption func(*CategoryService)

// WithCategoryEventSourcing enables the event-sourced write path for categories.
func WithCategoryEventSourcing(store eventstore.Store, registry *eventstore.ProjectorRegistry) CategoryServiceOption {
	return func(s *CategoryService) {
		if store != nil && registry != nil {
			s.evtStore = store
			s.projectors = registry
			s.eventsEnabled = true
		}
	}
}

// EventsEnabled reports whether the event-sourced write path is active.
func (s *CategoryService) EventsEnabled() bool { return s.eventsEnabled }

// WithCategoryEventObservability wires an EventObservability into the service.
func WithCategoryEventObservability(obs *eventstore.EventObservability) CategoryServiceOption {
	return func(s *CategoryService) {
		s.eventObs = obs
	}
}

type CreateCategoryInput struct {
	Name        string
	Description string
	Source      domain.CategorySource
}

type UpdateCategoryInput struct {
	ID          string
	Name        string
	Description string
}

func NewCategoryService(repo domain.CategoryRepository, opts ...CategoryServiceOption) *CategoryService {
	s := &CategoryService{repo: repo}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *CategoryService) List(ctx context.Context) ([]domain.Category, error) {
	return s.repo.List(ctx)
}

func (s *CategoryService) GetByID(ctx context.Context, id string) (*domain.Category, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("category id is required")
	}
	return s.repo.GetByID(ctx, id)
}

func (s *CategoryService) Create(ctx context.Context, input CreateCategoryInput) (domain.Category, error) {
	name := normalizeCategoryName(input.Name)
	if name == "" {
		return domain.Category{}, fmt.Errorf("category name is required")
	}

	existing, err := s.repo.GetByName(ctx, name)
	if err != nil {
		return domain.Category{}, err
	}
	if existing != nil {
		return *existing, nil
	}

	source := input.Source
	if source == "" {
		source = domain.CategorySourceManual
	}

	now := time.Now().UTC()
	category := domain.Category{
		ID:          uuid.NewString(),
		Name:        name,
		Description: strings.TrimSpace(input.Description),
		Source:      source,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if s.eventsEnabled {
		if err := s.createWithEvents(ctx, &category); err != nil {
			return domain.Category{}, err
		}
	} else {
		if err := s.repo.Create(ctx, &category); err != nil {
			return domain.Category{}, err
		}
	}

	return category, nil
}

func (s *CategoryService) EnsureByName(ctx context.Context, name string, source domain.CategorySource) (domain.Category, error) {
	normalized := normalizeCategoryName(name)
	if normalized == "" {
		return domain.Category{}, fmt.Errorf("category name is required")
	}

	existing, err := s.repo.GetByName(ctx, normalized)
	if err != nil {
		return domain.Category{}, err
	}
	if existing != nil {
		return *existing, nil
	}

	src := source
	if src == "" {
		src = domain.CategorySourceManual
	}
	now := time.Now().UTC()
	created := domain.Category{
		ID:        uuid.NewString(),
		Name:      normalized,
		Source:    src,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if s.eventsEnabled {
		if err := s.createWithEvents(ctx, &created); err != nil {
			return domain.Category{}, err
		}
	} else {
		if err := s.repo.Create(ctx, &created); err != nil {
			return domain.Category{}, err
		}
	}

	return created, nil
}

func (s *CategoryService) Update(ctx context.Context, input UpdateCategoryInput) (*domain.Category, error) {
	id := strings.TrimSpace(input.ID)
	if id == "" {
		return nil, fmt.Errorf("category id is required")
	}

	name := normalizeCategoryName(input.Name)
	if name == "" {
		return nil, fmt.Errorf("category name is required")
	}

	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, nil
	}

	byName, err := s.repo.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if byName != nil && byName.ID != existing.ID {
		return nil, fmt.Errorf("category name already exists")
	}

	existing.Name = name
	existing.Description = strings.TrimSpace(input.Description)
	existing.UpdatedAt = time.Now().UTC()

	if s.eventsEnabled {
		if err := s.updateWithEvents(ctx, existing); err != nil {
			return nil, err
		}
	} else {
		if err := s.repo.Update(ctx, existing); err != nil {
			return nil, err
		}
	}

	return existing, nil
}

func (s *CategoryService) Delete(ctx context.Context, id string) (bool, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return false, fmt.Errorf("category id is required")
	}

	existing, err := s.repo.GetByID(ctx, id)
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
		if err := s.repo.Delete(ctx, id); err != nil {
			return false, err
		}
	}

	return true, nil
}

// IncrementAccept and IncrementOverride are stat counters; not event-sourced.
func (s *CategoryService) IncrementAccept(ctx context.Context, id string) error {
	return s.repo.IncrementAccept(ctx, id)
}

func (s *CategoryService) IncrementOverride(ctx context.Context, id string) error {
	return s.repo.IncrementOverride(ctx, id)
}

// ── event-sourced write helpers ───────────────────────────────────────────────

func (s *CategoryService) createWithEvents(ctx context.Context, cat *domain.Category) error {
	payload, err := json.Marshal(eventstore.CategoryCreatedPayload{
		Name:        cat.Name,
		Description: cat.Description,
		Source:      string(cat.Source),
		CreatedAt:   cat.CreatedAt,
		UpdatedAt:   cat.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("marshal CategoryCreated payload: %w", err)
	}

	evt := eventstore.Event{
		EventID:       uuid.NewString(),
		AggregateID:   cat.ID,
		AggregateType: eventstore.AggregateTypeCategory,
		EventType:     eventstore.EventTypeCategoryCreated,
		EventVersion:  1,
		Payload:       json.RawMessage(payload),
	}

	committed, err := appendWithTx(ctx, s.evtStore, s.projectors, evt, s.eventObs)
	if err != nil {
		return err
	}
	s.projectors.ApplyAsync(ctx, committed)
	return nil
}

func (s *CategoryService) updateWithEvents(ctx context.Context, cat *domain.Category) error {
	payload, err := json.Marshal(eventstore.CategoryUpdatedPayload{
		Name:        cat.Name,
		Description: cat.Description,
		UpdatedAt:   cat.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("marshal CategoryUpdated payload: %w", err)
	}

	version, err := aggregateLatestVersion(ctx, s.evtStore, cat.ID)
	if err != nil {
		return err
	}

	evt := eventstore.Event{
		EventID:       uuid.NewString(),
		AggregateID:   cat.ID,
		AggregateType: eventstore.AggregateTypeCategory,
		EventType:     eventstore.EventTypeCategoryUpdated,
		EventVersion:  version + 1,
		Payload:       json.RawMessage(payload),
	}

	committed, err := appendWithTx(ctx, s.evtStore, s.projectors, evt, s.eventObs)
	if err != nil {
		return err
	}
	s.projectors.ApplyAsync(ctx, committed)
	return nil
}

func (s *CategoryService) deleteWithEvents(ctx context.Context, id string) error {
	payload, err := json.Marshal(eventstore.CategoryDeletedPayload{ID: id})
	if err != nil {
		return fmt.Errorf("marshal CategoryDeleted payload: %w", err)
	}

	version, err := aggregateLatestVersion(ctx, s.evtStore, id)
	if err != nil {
		return err
	}

	evt := eventstore.Event{
		EventID:       uuid.NewString(),
		AggregateID:   id,
		AggregateType: eventstore.AggregateTypeCategory,
		EventType:     eventstore.EventTypeCategoryDeleted,
		EventVersion:  version + 1,
		Payload:       json.RawMessage(payload),
	}

	committed, err := appendWithTx(ctx, s.evtStore, s.projectors, evt, s.eventObs)
	if err != nil {
		return err
	}
	s.projectors.ApplyAsync(ctx, committed)
	return nil
}

func normalizeCategoryName(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}

	fields := strings.Fields(trimmed)
	for i, field := range fields {
		runes := []rune(strings.ToLower(field))
		if len(runes) == 0 {
			continue
		}
		runes[0] = unicode.ToUpper(runes[0])
		fields[i] = string(runes)
	}
	return strings.Join(fields, " ")
}
