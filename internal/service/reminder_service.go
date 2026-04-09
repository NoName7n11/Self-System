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

func (s *ReminderService) List(ctx context.Context, limit, offset int) ([]domain.Reminder, error) {
	return s.repo.List(ctx, limit, offset)
}
