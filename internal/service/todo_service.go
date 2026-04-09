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

func (s *TodoService) List(ctx context.Context, limit, offset int) ([]domain.Todo, error) {
	return s.repo.List(ctx, limit, offset)
}
