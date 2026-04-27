package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"selfsystems/internal/domain"
)

type TodoService struct {
	repo domain.TodoRepository
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

func NewTodoService(repo domain.TodoRepository) *TodoService {
	return &TodoService{repo: repo}
}

func (s *TodoService) Create(ctx context.Context, input CreateTodoInput) (domain.Todo, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return domain.Todo{}, fmt.Errorf("todo title is required")
	}

	todo := domain.Todo{
		ID:         uuid.NewString(),
		Title:      title,
		Details:    strings.TrimSpace(input.Details),
		Status:     domain.TodoStatusOpen,
		DueAt:      input.DueAt,
		ResourceID: input.ResourceID,
	}
	if err := s.repo.Create(ctx, &todo); err != nil {
		return domain.Todo{}, err
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

	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, err
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

	if err := s.repo.Delete(ctx, id); err != nil {
		return false, err
	}

	return true, nil
}

func (s *TodoService) List(ctx context.Context, limit, offset int) ([]domain.Todo, error) {
	return s.repo.List(ctx, limit, offset)
}

func isValidTodoStatus(status domain.TodoStatus) bool {
	switch status {
	case domain.TodoStatusOpen, domain.TodoStatusInProgress, domain.TodoStatusDone:
		return true
	default:
		return false
	}
}
