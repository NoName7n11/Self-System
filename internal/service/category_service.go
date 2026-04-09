package service

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/google/uuid"

	"selfsystems/internal/domain"
)

type CategoryService struct {
	repo domain.CategoryRepository
}

type CreateCategoryInput struct {
	Name        string
	Description string
	Source      domain.CategorySource
}

func NewCategoryService(repo domain.CategoryRepository) *CategoryService {
	return &CategoryService{repo: repo}
}

func (s *CategoryService) List(ctx context.Context) ([]domain.Category, error) {
	return s.repo.List(ctx)
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

	category := domain.Category{
		ID:          uuid.NewString(),
		Name:        name,
		Description: strings.TrimSpace(input.Description),
		Source:      source,
	}

	if err := s.repo.Create(ctx, &category); err != nil {
		return domain.Category{}, err
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

	created := domain.Category{
		ID:     uuid.NewString(),
		Name:   normalized,
		Source: source,
	}
	if created.Source == "" {
		created.Source = domain.CategorySourceManual
	}
	if err := s.repo.Create(ctx, &created); err != nil {
		return domain.Category{}, err
	}

	return created, nil
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
