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

type ReminderService struct {
	repo          domain.ReminderRepository
	evtStore      eventstore.Store
	projectors    *eventstore.ProjectorRegistry
	eventsEnabled bool
	eventObs      *eventstore.EventObservability
}

// ReminderServiceOption configures a ReminderService.
type ReminderServiceOption func(*ReminderService)

// WithReminderEventSourcing enables the event-sourced write path for reminders.
func WithReminderEventSourcing(store eventstore.Store, registry *eventstore.ProjectorRegistry) ReminderServiceOption {
	return func(s *ReminderService) {
		if store != nil && registry != nil {
			s.evtStore = store
			s.projectors = registry
			s.eventsEnabled = true
		}
	}
}

// EventsEnabled reports whether the event-sourced write path is active.
func (s *ReminderService) EventsEnabled() bool { return s.eventsEnabled }

// WithReminderEventObservability wires an EventObservability into the service.
func WithReminderEventObservability(obs *eventstore.EventObservability) ReminderServiceOption {
	return func(s *ReminderService) {
		s.eventObs = obs
	}
}

type CreateReminderInput struct {
	Title      string
	Message    string
	RemindAt   time.Time
	ResourceID *string
}

type UpdateReminderInput struct {
	ID         string
	Title      string
	Message    string
	RemindAt   time.Time
	Status     domain.ReminderStatus
	ResourceID *string
}

func NewReminderService(repo domain.ReminderRepository, opts ...ReminderServiceOption) *ReminderService {
	s := &ReminderService{repo: repo}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *ReminderService) Create(ctx context.Context, input CreateReminderInput) (domain.Reminder, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return domain.Reminder{}, fmt.Errorf("reminder title is required")
	}
	if input.RemindAt.IsZero() {
		return domain.Reminder{}, fmt.Errorf("remind_at is required")
	}

	now := time.Now().UTC()
	reminder := domain.Reminder{
		ID:         uuid.NewString(),
		Title:      title,
		Message:    strings.TrimSpace(input.Message),
		RemindAt:   input.RemindAt.UTC(),
		Status:     domain.ReminderStatusScheduled,
		ResourceID: input.ResourceID,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if s.eventsEnabled {
		if err := s.createWithEvents(ctx, &reminder); err != nil {
			return domain.Reminder{}, err
		}
	} else {
		if err := s.repo.Create(ctx, &reminder); err != nil {
			return domain.Reminder{}, err
		}
	}

	return reminder, nil
}

func (s *ReminderService) GetByID(ctx context.Context, id string) (*domain.Reminder, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("reminder id is required")
	}
	return s.repo.GetByID(ctx, id)
}

func (s *ReminderService) Update(ctx context.Context, input UpdateReminderInput) (*domain.Reminder, error) {
	id := strings.TrimSpace(input.ID)
	if id == "" {
		return nil, fmt.Errorf("reminder id is required")
	}

	title := strings.TrimSpace(input.Title)
	if title == "" {
		return nil, fmt.Errorf("reminder title is required")
	}
	if input.RemindAt.IsZero() {
		return nil, fmt.Errorf("remind_at is required")
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
	if !isValidReminderStatus(status) {
		return nil, fmt.Errorf("reminder status is invalid")
	}

	existing.Title = title
	existing.Message = strings.TrimSpace(input.Message)
	existing.RemindAt = input.RemindAt.UTC()
	existing.Status = status
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

func (s *ReminderService) Delete(ctx context.Context, id string) (bool, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return false, fmt.Errorf("reminder id is required")
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

func (s *ReminderService) List(ctx context.Context, limit, offset int) ([]domain.Reminder, error) {
	return s.repo.List(ctx, limit, offset)
}

// ── event-sourced write helpers ───────────────────────────────────────────────

func (s *ReminderService) createWithEvents(ctx context.Context, r *domain.Reminder) error {
	payload, err := json.Marshal(eventstore.ReminderCreatedPayload{
		Title:      r.Title,
		Message:    r.Message,
		RemindAt:   r.RemindAt,
		Status:     string(r.Status),
		ResourceID: r.ResourceID,
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("marshal ReminderCreated payload: %w", err)
	}

	evt := eventstore.Event{
		EventID:       uuid.NewString(),
		AggregateID:   r.ID,
		AggregateType: eventstore.AggregateTypeReminder,
		EventType:     eventstore.EventTypeReminderCreated,
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

func (s *ReminderService) updateWithEvents(ctx context.Context, r *domain.Reminder) error {
	payload, err := json.Marshal(eventstore.ReminderUpdatedPayload{
		Title:      r.Title,
		Message:    r.Message,
		RemindAt:   r.RemindAt,
		Status:     string(r.Status),
		ResourceID: r.ResourceID,
		UpdatedAt:  r.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("marshal ReminderUpdated payload: %w", err)
	}

	version, err := aggregateLatestVersion(ctx, s.evtStore, r.ID)
	if err != nil {
		return err
	}

	evt := eventstore.Event{
		EventID:       uuid.NewString(),
		AggregateID:   r.ID,
		AggregateType: eventstore.AggregateTypeReminder,
		EventType:     eventstore.EventTypeReminderUpdated,
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

func (s *ReminderService) deleteWithEvents(ctx context.Context, id string) error {
	payload, err := json.Marshal(eventstore.ReminderDeletedPayload{ID: id})
	if err != nil {
		return fmt.Errorf("marshal ReminderDeleted payload: %w", err)
	}

	version, err := aggregateLatestVersion(ctx, s.evtStore, id)
	if err != nil {
		return err
	}

	evt := eventstore.Event{
		EventID:       uuid.NewString(),
		AggregateID:   id,
		AggregateType: eventstore.AggregateTypeReminder,
		EventType:     eventstore.EventTypeReminderDeleted,
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

func isValidReminderStatus(status domain.ReminderStatus) bool {
	switch status {
	case domain.ReminderStatusScheduled, domain.ReminderStatusSent, domain.ReminderStatusDismissed:
		return true
	default:
		return false
	}
}
