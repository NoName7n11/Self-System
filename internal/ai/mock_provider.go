package ai

import "context"

// MockProvider is an exported test double that satisfies Provider,
// EmbeddingProvider, and EnrichmentProvider. Register it on a Manager in
// integration tests to avoid real API calls.
//
// Usage:
//
//	mock := ai.NewMockProvider("mock")
//	mock.ClassifyOutput = ai.ClassificationOutput{SuggestedCategory: "Tech", Confidence: 0.95}
//	mock.EmbeddingOutput = ai.Embedding{Vector: make([]float32, 256), ModelVersion: "mock-v1", Dim: 256}
//	mock.EnrichOutput = ai.EnrichmentResult{Summary: "Test summary", KeyPoints: []string{"k1"}}
//	mgr := ai.NewManager("mock")
//	mgr.Register(mock)
//	mgr.RegisterEmbedding(mock)
//	mgr.RegisterEnrichment(mock)
type MockProvider struct {
	name string

	ClassifyOutput ClassificationOutput
	ClassifyErr    error

	EmbeddingOutput Embedding
	EmbeddingErr    error

	EnrichOutput EnrichmentResult
	EnrichErr    error

	// Counters incremented on each call — readable in tests.
	ClassifyCalls  int
	EmbeddingCalls int
	EnrichCalls    int
}

// NewMockProvider returns a MockProvider with the given name.
// Set the Output/Err fields before use.
func NewMockProvider(name string) *MockProvider {
	return &MockProvider{name: name}
}

func (m *MockProvider) Name() string { return m.name }

// ClassifySkim satisfies Provider.
func (m *MockProvider) ClassifySkim(_ context.Context, _ ClassificationInput) (ClassificationOutput, error) {
	m.ClassifyCalls++
	if m.ClassifyErr != nil {
		return ClassificationOutput{}, m.ClassifyErr
	}
	out := m.ClassifyOutput
	out.Provider = m.name
	return out, nil
}

// GenerateEmbedding satisfies EmbeddingProvider.
func (m *MockProvider) GenerateEmbedding(_ context.Context, _ string) (Embedding, error) {
	m.EmbeddingCalls++
	if m.EmbeddingErr != nil {
		return Embedding{}, m.EmbeddingErr
	}
	return m.EmbeddingOutput, nil
}

// Enrich satisfies EnrichmentProvider.
func (m *MockProvider) Enrich(_ context.Context, _ EnrichmentInput) (EnrichmentResult, error) {
	m.EnrichCalls++
	if m.EnrichErr != nil {
		return EnrichmentResult{}, m.EnrichErr
	}
	result := m.EnrichOutput
	result.Provider = m.name
	return result, nil
}
