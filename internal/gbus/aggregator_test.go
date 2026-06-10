package gbus

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"selfsystems/internal/eventstore"
)

// inMemoryFeatureStore is a simple in-memory FeatureStore for aggregator tests.
type inMemoryFeatureStore struct {
	catFeatures map[string]CategoryFeature  // key: categoryID+"|"+signalType
	resFeatures map[string]ResourceFeature  // key: resourceID+"|"+signalType
}

func newInMemoryFeatureStore() *inMemoryFeatureStore {
	return &inMemoryFeatureStore{
		catFeatures: map[string]CategoryFeature{},
		resFeatures: map[string]ResourceFeature{},
	}
}

func (s *inMemoryFeatureStore) UpsertCategoryFeature(_ context.Context, catID, sigType string, weight float64) error {
	key := catID + "|" + sigType
	f := s.catFeatures[key]
	f.CategoryID = catID
	f.SignalType = sigType
	f.TotalWeight += weight
	f.SignalCount++
	f.LastSignalAt = time.Now().UTC()
	s.catFeatures[key] = f
	return nil
}
func (s *inMemoryFeatureStore) GetCategoryFeatures(_ context.Context, catID string) ([]CategoryFeature, error) {
	var out []CategoryFeature
	for _, f := range s.catFeatures {
		if f.CategoryID == catID {
			out = append(out, f)
		}
	}
	return out, nil
}
func (s *inMemoryFeatureStore) ListAllCategoryFeatures(_ context.Context) ([]CategoryFeature, error) {
	out := make([]CategoryFeature, 0, len(s.catFeatures))
	for _, f := range s.catFeatures {
		out = append(out, f)
	}
	return out, nil
}
func (s *inMemoryFeatureStore) UpsertResourceFeature(_ context.Context, resID, sigType string, weight float64) error {
	key := resID + "|" + sigType
	f := s.resFeatures[key]
	f.ResourceID = resID
	f.SignalType = sigType
	f.TotalWeight += weight
	f.SignalCount++
	f.LastSignalAt = time.Now().UTC()
	s.resFeatures[key] = f
	return nil
}
func (s *inMemoryFeatureStore) GetResourceFeatures(_ context.Context, resID string) ([]ResourceFeature, error) {
	var out []ResourceFeature
	for _, f := range s.resFeatures {
		if f.ResourceID == resID {
			out = append(out, f)
		}
	}
	return out, nil
}
func (s *inMemoryFeatureStore) DeleteResourceFeatures(_ context.Context, _ string) error { return nil }
func (s *inMemoryFeatureStore) PruneOlderThan(_ context.Context, _ time.Time) (int, error) {
	return 0, nil
}

// buildSignalStore returns a stubStore populated with gbus signal events.
func buildSignalStore(payloads []GBUSSignalPayload) *stubStore {
	store := &stubStore{}
	for i, p := range payloads {
		if p.OccurredAt.IsZero() {
			p.OccurredAt = time.Now().UTC()
		}
		b, _ := json.Marshal(p)
		store.events = append(store.events, eventstore.Event{
			Sequence:      int64(i + 1),
			EventID:       uuid.NewString(),
			AggregateID:   uuid.NewString(),
			AggregateType: AggregateTypeGBUS,
			EventType:     EventTypeGBUSBase + "." + p.SignalType,
			EventVersion:  1,
			Payload:       b,
		})
	}
	return store
}

func TestAggregator_ProcessesCategorySignals(t *testing.T) {
	eventStore := buildSignalStore([]GBUSSignalPayload{
		{SignalType: SignalManualClassification, CategoryID: "cat-1", ResourceID: "res-1", Weight: 1.0},
		{SignalType: SignalManualClassification, CategoryID: "cat-1", ResourceID: "res-2", Weight: 1.0},
		{SignalType: SignalAutoClassification, CategoryID: "cat-2", ResourceID: "res-3", Weight: 0.5},
	})
	features := newInMemoryFeatureStore()
	agg := NewAggregator(eventStore, features, 0)

	n, err := agg.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if n != 3 {
		t.Errorf("processed = %d, want 3", n)
	}

	cat1Features, _ := features.GetCategoryFeatures(context.Background(), "cat-1")
	if len(cat1Features) != 1 {
		t.Fatalf("cat-1 feature rows = %d, want 1", len(cat1Features))
	}
	if cat1Features[0].TotalWeight != 2.0 {
		t.Errorf("cat-1 total_weight = %v, want 2.0", cat1Features[0].TotalWeight)
	}
	if cat1Features[0].SignalCount != 2 {
		t.Errorf("cat-1 signal_count = %d, want 2", cat1Features[0].SignalCount)
	}
}

func TestAggregator_ProcessesResourceSignals(t *testing.T) {
	eventStore := buildSignalStore([]GBUSSignalPayload{
		{SignalType: SignalResourceSaved, ResourceID: "res-1", Weight: 0.3},
		{SignalType: SignalResourceDeleted, ResourceID: "res-1", Weight: 0.1},
	})
	features := newInMemoryFeatureStore()
	agg := NewAggregator(eventStore, features, 0)

	if _, err := agg.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	resFeatures, _ := features.GetResourceFeatures(context.Background(), "res-1")
	if len(resFeatures) != 2 {
		t.Errorf("res-1 feature rows = %d, want 2", len(resFeatures))
	}
}

func TestAggregator_SkipsNonGBUSEvents(t *testing.T) {
	store := &stubStore{}
	store.events = append(store.events, eventstore.Event{
		Sequence:      1,
		EventID:       uuid.NewString(),
		AggregateID:   uuid.NewString(),
		AggregateType: "resource", // not gbus_signal
		EventType:     "ResourceCreated",
		EventVersion:  1,
		Payload:       []byte(`{"url":"https://example.com"}`),
	})
	features := newInMemoryFeatureStore()
	agg := NewAggregator(store, features, 0)

	n, err := agg.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if n != 0 {
		t.Errorf("non-gbus event should not be counted; got %d", n)
	}
	all, _ := features.ListAllCategoryFeatures(context.Background())
	if len(all) != 0 {
		t.Errorf("expected no category features for non-gbus event")
	}
}

func TestAggregator_SequenceTracking(t *testing.T) {
	store := buildSignalStore([]GBUSSignalPayload{
		{SignalType: SignalResourceSaved, ResourceID: "res-1", Weight: 0.3},
	})
	features := newInMemoryFeatureStore()
	agg := NewAggregator(store, features, 0)

	// First run processes the event.
	n, _ := agg.Run(context.Background())
	if n != 1 {
		t.Errorf("first run: processed = %d, want 1", n)
	}

	// Second run: stub returns same events (seq >= 1), but aggregator starts
	// after lastSequence=1 so ReadBySequence is called with afterSequence=1.
	// The stub always returns all events regardless of afterSequence, so to
	// simulate incremental reads we clear the store.
	store.events = nil
	n2, _ := agg.Run(context.Background())
	if n2 != 0 {
		t.Errorf("second run on empty store: processed = %d, want 0", n2)
	}
}
