package service

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"

	"selfsystems/internal/domain"
)

type ResourceService struct {
	resources   domain.ResourceRepository
	categories  domain.CategoryRepository
	classifier  *CategoryClassifier
	catSvc      *CategoryService
}

type CreateResourceInput struct {
	URL          string
	Title        string
	Summary      string
	CategoryID   string
	CategoryName string
}

type UpdateResourceCategoryInput struct {
	ResourceID string
	CategoryID string
}

func NewResourceService(
	resources domain.ResourceRepository,
	categories domain.CategoryRepository,
	classifier *CategoryClassifier,
	catSvc *CategoryService,
) *ResourceService {
	return &ResourceService{
		resources:  resources,
		categories: categories,
		classifier: classifier,
		catSvc:     catSvc,
	}
}

func (s *ResourceService) Create(ctx context.Context, input CreateResourceInput) (domain.Resource, error) {
	normalizedURL, host, err := normalizeURL(input.URL)
	if err != nil {
		return domain.Resource{}, err
	}

	var category domain.Category
	userOverride := false

	if strings.TrimSpace(input.CategoryID) != "" {
		categoryPtr, err := s.categories.GetByID(ctx, strings.TrimSpace(input.CategoryID))
		if err != nil {
			return domain.Resource{}, err
		}
		if categoryPtr == nil {
			return domain.Resource{}, fmt.Errorf("category not found")
		}
		category = *categoryPtr
		userOverride = true
	} else if strings.TrimSpace(input.CategoryName) != "" {
		category, err = s.catSvc.EnsureByName(ctx, input.CategoryName, domain.CategorySourceManual)
		if err != nil {
			return domain.Resource{}, err
		}
		userOverride = true
	} else {
		suggestion, err := s.classifier.Suggest(ctx, normalizedURL, input.Title)
		if err != nil {
			return domain.Resource{}, err
		}
		category = suggestion.Category
	}

	resource := domain.Resource{
		ID:           uuid.NewString(),
		URL:          normalizedURL,
		Host:         host,
		Title:        strings.TrimSpace(input.Title),
		Summary:      strings.TrimSpace(input.Summary),
		CategoryID:   category.ID,
		CategoryName: category.Name,
		UserOverride: userOverride,
	}

	if resource.Title == "" {
		resource.Title = inferTitleFromURL(normalizedURL)
	}

	if err := s.resources.Create(ctx, &resource); err != nil {
		return domain.Resource{}, err
	}

	if err := s.categories.IncrementAccept(ctx, category.ID); err != nil {
		return domain.Resource{}, err
	}

	return resource, nil
}

func (s *ResourceService) List(ctx context.Context, limit, offset int) ([]domain.Resource, error) {
	return s.resources.List(ctx, limit, offset)
}

func (s *ResourceService) Search(ctx context.Context, query string, limit int) ([]domain.Resource, error) {
	return s.resources.Search(ctx, query, limit)
}

func (s *ResourceService) UpdateCategory(ctx context.Context, input UpdateResourceCategoryInput) error {
	if strings.TrimSpace(input.ResourceID) == "" || strings.TrimSpace(input.CategoryID) == "" {
		return fmt.Errorf("resource_id and category_id are required")
	}

	category, err := s.categories.GetByID(ctx, input.CategoryID)
	if err != nil {
		return err
	}
	if category == nil {
		return fmt.Errorf("category not found")
	}

	if err := s.resources.UpdateCategory(ctx, input.ResourceID, input.CategoryID, true); err != nil {
		return err
	}

	if err := s.categories.IncrementAccept(ctx, input.CategoryID); err != nil {
		return err
	}

	return nil
}

func normalizeURL(rawURL string) (string, string, error) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return "", "", fmt.Errorf("url is required")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", "", fmt.Errorf("invalid url")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", "", fmt.Errorf("url must start with http or https")
	}

	host := strings.TrimPrefix(parsed.Hostname(), "www.")
	return parsed.String(), host, nil
}

func inferTitleFromURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "Untitled Resource"
	}
	host := strings.TrimPrefix(parsed.Hostname(), "www.")
	if host == "" {
		return "Untitled Resource"
	}
	return normalizeCategoryName(strings.ReplaceAll(host, ".", " "))
}
