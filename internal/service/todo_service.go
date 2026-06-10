package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"selfsystems/internal/domain"
	"selfsystems/internal/eventstore"
)

type TodoService struct {
	repo          domain.TodoRepository
	evtStore      eventstore.Store
	projectors    *eventstore.ProjectorRegistry
	eventsEnabled bool
	eventObs      *eventstore.EventObservability
}

// TodoServiceOption configures a TodoService.
type TodoServiceOption func(*TodoService)

// WithTodoEventSourcing enables the event-sourced write path for todos.
func WithTodoEventSourcing(store eventstore.Store, registry *eventstore.ProjectorRegistry) TodoServiceOption {
	return func(s *TodoService) {
		if store != nil && registry != nil {
			s.evtStore = store
			s.projectors = registry
			s.eventsEnabled = true
		}
	}
}

// EventsEnabled reports whether the event-sourced write path is active.
func (s *TodoService) EventsEnabled() bool { return s.eventsEnabled }

// WithTodoEventObservability wires an EventObservability into the service.
func WithTodoEventObservability(obs *eventstore.EventObservability) TodoServiceOption {
	return func(s *TodoService) {
		s.eventObs = obs
	}
}

type CreateTodoInput struct {
	Title      string
	Details    string
	DueAt      *time.Time
	ResourceID *string
}

type UpdateTodoInput struct {
	ID         string
	Title      string
	Details    string
	Status     domain.TodoStatus
	DueAt      *time.Time
	ResourceID *string
}

func NewTodoService(repo domain.TodoRepository, opts ...TodoServiceOption) *TodoService {
	s := &TodoService{repo: repo}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *TodoService) Create(ctx context.Context, input CreateTodoInput) (domain.Todo, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return domain.Todo{}, fmt.Errorf("todo title is required")
	}

	now := time.Now().UTC()
	todo := domain.Todo{
		ID:         uuid.NewString(),
		Title:      title,
		Details:    strings.TrimSpace(input.Details),
		Status:     domain.TodoStatusOpen,
		DueAt:      input.DueAt,
		ResourceID: input.ResourceID,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if s.eventsEnabled {
		if err := s.createWithEvents(ctx, &todo); err != nil {
			return domain.Todo{}, err
		}
	} else {
		if err := s.repo.Create(ctx, &todo); err != nil {
			return domain.Todo{}, err
		}
	}
	return todo, nil
}

func (s *TodoService) GetByID(ctx context.Context, id string) (*domain.Todo, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("todo id is required")
	}
	return s.repo.GetByID(ctx, id)
}

func (s *TodoService) Update(ctx context.Context, input UpdateTodoInput) (*domain.Todo, error) {
	id := strings.TrimSpace(input.ID)
	if id == "" {
		return nil, fmt.Errorf("todo id is required")
	}

	title := strings.TrimSpace(input.Title)
	if title == "" {
		return nil, fmt.Errorf("todo title is required")
	}

	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, nil
	}

	status := input.Status
	if status == "" {
		status = existing.Status
	}
	if !isValidTodoStatus(status) {
		return nil, fmt.Errorf("todo status is invalid")
	}

	existing.Title = title
	existing.Details = strings.TrimSpace(input.Details)
	existing.Status = status
	existing.DueAt = input.DueAt
	existing.ResourceID = input.ResourceID
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

func (s *TodoService) Delete(ctx context.Context, id string) (bool, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return false, fmt.Errorf("todo id is required")
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

func (s *TodoService) List(ctx context.Context, limit, offset int) ([]domain.Todo, error) {
	return s.repo.List(ctx, limit, offset)
}

// ── event-sourced write helpers ───────────────────────────────────────────────

func (s *TodoService) createWithEvents(ctx context.Context, todo *domain.Todo) error {
	payload, err := json.Marshal(eventstore.TodoCreatedPayload{
		Title:      todo.Title,
		Details:    todo.Details,
		Status:     string(todo.Status),
		DueAt:      todo.DueAt,
		ResourceID: todo.ResourceID,
		CreatedAt:  todo.CreatedAt,
		UpdatedAt:  todo.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("marshal TodoCreated payload: %w", err)
	}

	evt := eventstore.Event{
		EventID:       uuid.NewString(),
		AggregateID:   todo.ID,
		AggregateType: eventstore.AggregateTypeTodo,
		EventType:     eventstore.EventTypeTodoCreated,
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

func (s *TodoService) updateWithEvents(ctx context.Context, todo *domain.Todo) error {
	payload, err := json.Marshal(eventstore.TodoUpdatedPayload{
		Title:      todo.Title,
		Details:    todo.Details,
		Status:     string(todo.Status),
		DueAt:      todo.DueAt,
		ResourceID: todo.ResourceID,
		UpdatedAt:  todo.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("marshal TodoUpdated payload: %w", err)
	}

	version, err := aggregateLatestVersion(ctx, s.evtStore, todo.ID)
	if err != nil {
		return err
	}

	evt := eventstore.Event{
		EventID:       uuid.NewString(),
		AggregateID:   todo.ID,
		AggregateType: eventstore.AggregateTypeTodo,
		EventType:     eventstore.EventTypeTodoUpdated,
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

func (s *TodoService) deleteWithEvents(ctx context.Context, id string) error {
	payload, err := json.Marshal(eventstore.TodoDeletedPayload{ID: id})
	if err != nil {
		return fmt.Errorf("marshal TodoDeleted payload: %w", err)
	}

	version, err := aggregateLatestVersion(ctx, s.evtStore, id)
	if err != nil {
		return err
	}

	evt := eventstore.Event{
		EventID:       uuid.NewString(),
		AggregateID:   id,
		AggregateType: eventstore.AggregateTypeTodo,
		EventType:     eventstore.EventTypeTodoDeleted,
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

func isValidTodoStatus(status domain.TodoStatus) bool {
	switch status {
	case domain.TodoStatusOpen, domain.TodoStatusInProgress, domain.TodoStatusDone:
		return true
	default:
		return false
	}
}
