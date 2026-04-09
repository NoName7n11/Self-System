package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

func buildClassificationPrompt(input ClassificationInput) string {
	existing := "(none)"
	if len(input.ExistingCategories) > 0 {
		existing = strings.Join(input.ExistingCategories, ", ")
	}

	return fmt.Sprintf(
		"Classify this resource into exactly one category. If an existing category matches, prefer it. "+
			"Return strict JSON only: {\\\"category\\\":\\\"...\\\",\\\"confidence\\\":0.0-1.0,\\\"reason\\\":\\\"...\\\"}.\n"+
			"URL: %s\nTitle: %s\nSummary: %s\nExisting Categories: %s",
		input.URL,
		input.Title,
		input.Summary,
		existing,
	)
}

func parseModelOutput(raw string) (ClassificationOutput, error) {
	jsonPayload := extractJSONObject(raw)
	if jsonPayload == "" {
		return ClassificationOutput{}, fmt.Errorf("model response did not contain JSON object")
	}

	var parsed struct {
		Category   string  `json:"category"`
		Confidence float64 `json:"confidence"`
		Reason     string  `json:"reason"`
	}
	if err := json.Unmarshal([]byte(jsonPayload), &parsed); err != nil {
		return ClassificationOutput{}, fmt.Errorf("parse model json: %w", err)
	}

	category := normalizeCategoryName(parsed.Category)
	if category == "" {
		return ClassificationOutput{}, fmt.Errorf("model response missing category")
	}

	confidence := parsed.Confidence
	if confidence <= 0 {
		confidence = 0.65
	}
	if confidence > 1 {
		confidence = 1
	}

	reason := strings.TrimSpace(parsed.Reason)
	if reason == "" {
		reason = "AI provider suggested category"
	}

	return ClassificationOutput{
		SuggestedCategory: category,
		Confidence:        confidence,
		Reason:            reason,
	}, nil
}

func extractJSONObject(raw string) string {
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start == -1 || end == -1 || end < start {
		return ""
	}
	return strings.TrimSpace(raw[start : end+1])
}
