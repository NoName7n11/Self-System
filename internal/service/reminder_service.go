package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"selfsystems/internal/domain"
)

type ReminderService struct {
	repo domain.ReminderRepository
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

func NewReminderService(repo domain.ReminderRepository) *ReminderService {
	return &ReminderService{repo: repo}
}

func (s *ReminderService) Create(ctx context.Context, input CreateReminderInput) (domain.Reminder, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return domain.Reminder{}, fmt.Errorf("reminder title is required")
	}
	if input.RemindAt.IsZero() {
		return domain.Reminder{}, fmt.Errorf("remind_at is required")
	}

	reminder := domain.Reminder{
		ID:         uuid.NewString(),
		Title:      title,
		Message:    strings.TrimSpace(input.Message),
		RemindAt:   input.RemindAt.UTC(),
		Status:     domain.ReminderStatusScheduled,
		ResourceID: input.ResourceID,
	}
	if err := s.repo.Create(ctx, &reminder); err != nil {
		return domain.Reminder{}, err
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

	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, err
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

	if err := s.repo.Delete(ctx, id); err != nil {
		return false, err
	}

	return true, nil
}

func (s *ReminderService) List(ctx context.Context, limit, offset int) ([]domain.Reminder, error) {
	return s.repo.List(ctx, limit, offset)
}

func isValidReminderStatus(status domain.ReminderStatus) bool {
	switch status {
	case domain.ReminderStatusScheduled, domain.ReminderStatusSent, domain.ReminderStatusDismissed:
		return true
	default:
		return false
	}
}
