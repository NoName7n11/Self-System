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

type UpdateCategoryInput struct {
	ID          string
	Name        string
	Description string
}

func NewCategoryService(repo domain.CategoryRepository) *CategoryService {
	return &CategoryService{repo: repo}
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

	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, err
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

	if err := s.repo.Delete(ctx, id); err != nil {
		return false, err
	}

	return true, nil
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
