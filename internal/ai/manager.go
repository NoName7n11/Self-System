package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type Manager struct {
	primary   string
	fallback  string
	providers map[string]Provider
	order     []string
}

func NewManager(primary string) *Manager {
	return &Manager{
		primary:   normalizeProviderName(primary),
		providers: map[string]Provider{},
		order:     make([]string, 0),
	}
}

func (m *Manager) Register(provider Provider) {
	if provider == nil {
		return
	}
	name := normalizeProviderName(provider.Name())
	if name == "" {
		return
	}
	if _, exists := m.providers[name]; !exists {
		m.order = append(m.order, name)
	}
	m.providers[name] = provider
}

func (m *Manager) SetFallback(providerName string) {
	m.fallback = normalizeProviderName(providerName)
}

func (m *Manager) ClassifySkim(ctx context.Context, input ClassificationInput) (ClassificationOutput, error) {
	attempted := map[string]struct{}{}
	var lastErr error

	try := func(name string) (ClassificationOutput, bool) {
		name = normalizeProviderName(name)
		if name == "" {
			return ClassificationOutput{}, false
		}
		provider, exists := m.providers[name]
		if !exists {
			return ClassificationOutput{}, false
		}
		if _, seen := attempted[name]; seen {
			return ClassificationOutput{}, false
		}
		attempted[name] = struct{}{}

		output, err := provider.ClassifySkim(ctx, input)
		if err != nil {
			if errors.Is(err, ErrProviderUnavailable) {
				return ClassificationOutput{}, false
			}
			lastErr = fmt.Errorf("%s provider failed: %w", name, err)
			return ClassificationOutput{}, false
		}

		output.SuggestedCategory = strings.TrimSpace(output.SuggestedCategory)
		if output.SuggestedCategory == "" {
			return ClassificationOutput{}, false
		}
		if output.Confidence <= 0 {
			output.Confidence = 0.65
		}
		if output.Confidence > 1 {
			output.Confidence = 1
		}
		if output.Reason == "" {
			output.Reason = "AI provider suggested category"
		}
		return output, true
	}

	if output, ok := try(m.primary); ok {
		return output, nil
	}

	for _, name := range m.order {
		if name == m.primary || name == m.fallback {
			continue
		}
		if output, ok := try(name); ok {
			return output, nil
		}
	}

	if output, ok := try(m.fallback); ok {
		return output, nil
	}

	if lastErr != nil {
		return ClassificationOutput{}, lastErr
	}
	return ClassificationOutput{}, fmt.Errorf("no AI provider produced a classification")
}

func normalizeProviderName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
