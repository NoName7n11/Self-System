package service

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/google/uuid"

	"selfsystems/internal/domain"
)

type ResourceService struct {
	resources  domain.ResourceRepository
	categories domain.CategoryRepository
	classifier *CategoryClassifier
	catSvc     *CategoryService
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

func (s *ResourceService) SemanticSearch(ctx context.Context, query string, limit int) ([]domain.Resource, error) {
	trimmedQuery := strings.TrimSpace(query)
	if trimmedQuery == "" {
		return []domain.Resource{}, nil
	}

	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	queryTokens := semanticExpandTokens(semanticTokenize(trimmedQuery))
	if len(queryTokens) == 0 {
		return []domain.Resource{}, nil
	}

	candidates, err := s.resources.List(ctx, 500, 0)
	if err != nil {
		return nil, err
	}

	type scoredResource struct {
		resource domain.Resource
		score    float64
	}

	scored := make([]scoredResource, 0, len(candidates))
	for _, candidate := range candidates {
		score := semanticScore(trimmedQuery, queryTokens, candidate)
		if score >= 0.08 {
			scored = append(scored, scoredResource{resource: candidate, score: score})
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return scored[i].resource.CreatedAt.After(scored[j].resource.CreatedAt)
		}
		return scored[i].score > scored[j].score
	})

	results := make([]domain.Resource, 0, limit)
	for idx, item := range scored {
		if idx >= limit {
			break
		}
		results = append(results, item.resource)
	}

	return results, nil
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

var semanticTokenCleaner = regexp.MustCompile(`[^a-z0-9]+`)

var semanticSynonyms = map[string][]string{
	"ai":           {"artificial", "intelligence", "llm", "ml"},
	"llm":          {"ai", "language", "model"},
	"ml":           {"machine", "learning", "ai"},
	"agent":        {"assistant", "automation"},
	"agents":       {"assistant", "automation", "agent"},
	"graph":        {"network", "node", "relationship"},
	"graphs":       {"network", "node", "relationship", "graph"},
	"knowledge":    {"memory", "information"},
	"productivity": {"workflow", "task", "todo"},
	"todo":         {"task", "checklist"},
	"research":     {"study", "analysis"},
}

func semanticTokenize(input string) []string {
	cleaned := strings.ToLower(semanticTokenCleaner.ReplaceAllString(input, " "))
	parts := strings.Fields(cleaned)
	if len(parts) == 0 {
		return []string{}
	}
	return parts
}

func semanticExpandTokens(tokens []string) map[string]struct{} {
	set := map[string]struct{}{}
	for _, token := range tokens {
		trimmed := strings.TrimSpace(token)
		if trimmed == "" {
			continue
		}
		set[trimmed] = struct{}{}
		if synonyms, ok := semanticSynonyms[trimmed]; ok {
			for _, synonym := range synonyms {
				set[synonym] = struct{}{}
			}
		}
	}
	return set
}

func semanticScore(query string, expandedQueryTokens map[string]struct{}, resource domain.Resource) float64 {
	if len(expandedQueryTokens) == 0 {
		return 0
	}

	resourceText := strings.ToLower(strings.Join([]string{
		resource.Title,
		resource.Summary,
		resource.URL,
		resource.CategoryName,
	}, " "))

	resourceTokens := semanticExpandTokens(semanticTokenize(resourceText))
	if len(resourceTokens) == 0 {
		return 0
	}

	matches := 0
	for token := range expandedQueryTokens {
		if _, ok := resourceTokens[token]; ok {
			matches++
		}
	}

	coverage := float64(matches) / float64(len(expandedQueryTokens))
	score := coverage * 0.85

	trimmedQuery := strings.TrimSpace(strings.ToLower(query))
	if trimmedQuery != "" {
		if strings.Contains(strings.ToLower(resource.Title), trimmedQuery) || strings.Contains(strings.ToLower(resource.Summary), trimmedQuery) {
			score += 0.15
		}
	}

	if score > 1 {
		return 1
	}
	return score
}
