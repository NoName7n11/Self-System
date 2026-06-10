package ai

import (
	"context"
	"errors"
)

var ErrProviderUnavailable = errors.New("provider unavailable")

type ClassificationInput struct {
	URL                string
	Title              string
	Summary            string
	ExistingCategories []string
}

type ClassificationOutput struct {
	SuggestedCategory string
	Confidence        float64
	Reason            string
	// Provider is the name of the provider that produced this output
	// (e.g. "openai", "anthropic", "heuristic"). Stamped by the Manager.
	Provider string
}

type Provider interface {
	Name() string
	ClassifySkim(ctx context.Context, input ClassificationInput) (ClassificationOutput, error)
}

type ProviderSettings struct {
	Enabled        bool
	APIKey         string
	Model          string
	BaseURL        string
	TimeoutSeconds int
}
