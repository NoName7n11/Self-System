package sync

import "testing"

func TestWSHandlerClientConnectionLimit(t *testing.T) {
	handler := NewWSHandler(nil, nil, 30, 1, NewObservability())

	if !handler.acquireClientConnection("127.0.0.1") {
		t.Fatalf("expected first connection to be allowed")
	}
	if handler.acquireClientConnection("127.0.0.1") {
		t.Fatalf("expected second connection to be rejected when max is 1")
	}

	handler.releaseClientConnection("127.0.0.1")
	if !handler.acquireClientConnection("127.0.0.1") {
		t.Fatalf("expected connection to be allowed after release")
	}
}
