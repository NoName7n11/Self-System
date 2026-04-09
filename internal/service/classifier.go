package service

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"selfsystems/internal/domain"
)

type CategorySuggestion struct {
	Category    domain.Category
	Score       float64
	Reason      string
	AutoCreated bool
}

type CategoryClassifier struct {
	categories domain.CategoryRepository
}

func NewCategoryClassifier(categories domain.CategoryRepository) *CategoryClassifier {
	return &CategoryClassifier{categories: categories}
}

func (c *CategoryClassifier) Suggest(ctx context.Context, rawURL, title string) (CategorySuggestion, error) {
	items, err := c.categories.List(ctx)
	if err != nil {
		return CategorySuggestion{}, fmt.Errorf("list categories for classifier: %w", err)
	}

	tokens := tokenize(rawURL + " " + title)
	if len(items) == 0 {
		categoryName := deriveCategoryName(rawURL, title)
		category := domain.Category{
			ID:     uuid.NewString(),
			Name:   categoryName,
			Source: domain.CategorySourceAuto,
		}
		if err := c.categories.Create(ctx, &category); err != nil {
			return CategorySuggestion{}, err
		}
		return CategorySuggestion{
			Category:    category,
			Score:       1.0,
			Reason:      "No categories existed, auto-created initial category",
			AutoCreated: true,
		}, nil
	}

	best := items[0]
	bestScore := scoreCategory(best, tokens)
	for _, item := range items[1:] {
		score := scoreCategory(item, tokens)
		if score > bestScore {
			best = item
			bestScore = score
		}
	}

	if bestScore < 0.55 {
		categoryName := deriveCategoryName(rawURL, title)
		existing, err := c.categories.GetByName(ctx, categoryName)
		if err != nil {
			return CategorySuggestion{}, err
		}
		if existing != nil {
			return CategorySuggestion{
				Category:    *existing,
				Score:       bestScore,
				Reason:      "Low confidence on existing categories, using matching auto category",
				AutoCreated: false,
			}, nil
		}

		category := domain.Category{
			ID:     uuid.NewString(),
			Name:   categoryName,
			Source: domain.CategorySourceAuto,
		}
		if err := c.categories.Create(ctx, &category); err != nil {
			return CategorySuggestion{}, err
		}
		return CategorySuggestion{
			Category:    category,
			Score:       0.75,
			Reason:      "Classifier confidence was low; auto-created category based on URL context",
			AutoCreated: true,
		}, nil
	}

	return CategorySuggestion{
		Category:    best,
		Score:       bestScore,
		Reason:      "Best score among existing categories",
		AutoCreated: false,
	}, nil
}

func scoreCategory(category domain.Category, tokens []string) float64 {
	name := strings.ToLower(category.Name)
	score := 0.0

	for _, token := range tokens {
		if token == "" {
			continue
		}
		if token == name {
			score += 1.25
			continue
		}
		if strings.Contains(name, token) || strings.Contains(token, name) {
			score += 0.9
		}
	}

	score += float64(category.AcceptCount) * 0.03
	score -= float64(category.OverrideCount) * 0.015
	return score
}

func deriveCategoryName(rawURL, title string) string {
	combined := strings.ToLower(rawURL + " " + title)
	keywordMap := map[string]string{
		"github":      "Development",
		"dev":         "Development",
		"programming": "Development",
		"code":        "Development",
		"ai":          "AI",
		"llm":         "AI",
		"machine":     "AI",
		"learning":    "AI",
		"news":        "News",
		"research":    "Research",
		"productivity": "Productivity",
		"finance":     "Finance",
		"health":      "Health",
		"design":      "Design",
		"video":       "Video",
		"youtube":     "Video",
	}
	for keyword, category := range keywordMap {
		if strings.Contains(combined, keyword) {
			return category
		}
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "General"
	}
	host := strings.TrimPrefix(parsed.Hostname(), "www.")
	if host == "" {
		return "General"
	}
	parts := strings.Split(host, ".")
	if len(parts) > 0 {
		return normalizeCategoryName(parts[0])
	}
	return "General"
}

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)

func tokenize(input string) []string {
	cleaned := strings.ToLower(nonAlphaNum.ReplaceAllString(input, " "))
	parts := strings.Fields(cleaned)
	if len(parts) == 0 {
		return []string{"general"}
	}
	return parts
}
