package sync

import "testing"

func TestInferEntityIDDoesNotFallbackToGenericID(t *testing.T) {
	id := inferEntityID(map[string]any{"id": "res-1"})
	if id != "" {
		t.Fatalf("expected empty entity id when only generic id key exists, got %q", id)
	}
}
