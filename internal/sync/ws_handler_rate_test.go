package sync

import (
	"testing"
	"time"
)

func TestInboundMessageLimiterAllowsWithinLimit(t *testing.T) {
	limiter := newInboundMessageLimiter(3, time.Minute)
	start := time.Now().UTC()

	if !limiter.Allow(start) {
		t.Fatalf("expected first message to be allowed")
	}
	if !limiter.Allow(start.Add(10 * time.Second)) {
		t.Fatalf("expected second message to be allowed")
	}
	if !limiter.Allow(start.Add(20 * time.Second)) {
		t.Fatalf("expected third message to be allowed")
	}
}

func TestInboundMessageLimiterRejectsOverLimit(t *testing.T) {
	limiter := newInboundMessageLimiter(2, time.Minute)
	start := time.Now().UTC()

	if !limiter.Allow(start) {
		t.Fatalf("expected first message to be allowed")
	}
	if !limiter.Allow(start.Add(1 * time.Second)) {
		t.Fatalf("expected second message to be allowed")
	}
	if limiter.Allow(start.Add(2 * time.Second)) {
		t.Fatalf("expected third message to be rejected")
	}
}

func TestInboundMessageLimiterResetsAfterWindow(t *testing.T) {
	limiter := newInboundMessageLimiter(1, time.Minute)
	start := time.Now().UTC()

	if !limiter.Allow(start) {
		t.Fatalf("expected first message to be allowed")
	}
	if limiter.Allow(start.Add(10 * time.Second)) {
		t.Fatalf("expected second message in same window to be rejected")
	}
	if !limiter.Allow(start.Add(61 * time.Second)) {
		t.Fatalf("expected message after window reset to be allowed")
	}
}
