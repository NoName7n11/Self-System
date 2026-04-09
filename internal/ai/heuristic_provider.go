package ai

import (
	"context"
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

type HeuristicProvider struct{}

func NewHeuristicProvider() *HeuristicProvider {
	return &HeuristicProvider{}
}

func (p *HeuristicProvider) Name() string {
	return "heuristic"
}

func (p *HeuristicProvider) ClassifySkim(_ context.Context, input ClassificationInput) (ClassificationOutput, error) {
	tokens := tokenize(input.URL + " " + input.Title + " " + input.Summary)

	bestCategory := ""
	bestScore := 0.0
	for _, category := range input.ExistingCategories {
		score := scoreExistingCategory(category, tokens)
		if score > bestScore {
			bestScore = score
			bestCategory = category
		}
	}

	if bestCategory != "" && bestScore >= 0.55 {
		confidence := bestScore
		if confidence > 0.95 {
			confidence = 0.95
		}
		return ClassificationOutput{
			SuggestedCategory: normalizeCategoryName(bestCategory),
			Confidence:        confidence,
			Reason:            "Matched existing category by skim keyword scoring",
		}, nil
	}

	derived := deriveCategoryName(input.URL, input.Title, input.Summary)
	return ClassificationOutput{
		SuggestedCategory: derived,
		Confidence:        0.62,
		Reason:            "Derived category from URL/title heuristics",
	}, nil
}

func scoreExistingCategory(category string, tokens []string) float64 {
	name := strings.ToLower(strings.TrimSpace(category))
	if name == "" {
		return 0
	}

	score := 0.0
	for _, token := range tokens {
		if token == "" {
			continue
		}
		if token == name {
			score += 1.2
			continue
		}
		if strings.Contains(name, token) || strings.Contains(token, name) {
			score += 0.75
		}
	}
	if score == 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}

func deriveCategoryName(rawURL, title, summary string) string {
	combined := strings.ToLower(rawURL + " " + title + " " + summary)
	keywordMap := map[string]string{
		"github":       "Development",
		"programming":  "Development",
		"code":         "Development",
		"ai":           "AI",
		"llm":          "AI",
		"machine":      "AI",
		"learning":     "AI",
		"news":         "News",
		"research":     "Research",
		"productivity": "Productivity",
		"finance":      "Finance",
		"health":       "Health",
		"design":       "Design",
		"video":        "Video",
		"youtube":      "Video",
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

var tokenCleaner = regexp.MustCompile(`[^a-z0-9]+`)

func tokenize(input string) []string {
	cleaned := strings.ToLower(tokenCleaner.ReplaceAllString(input, " "))
	parts := strings.Fields(cleaned)
	if len(parts) == 0 {
		return []string{"general"}
	}
	return parts
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
