package service

import "testing"

func TestParsePipePayload(t *testing.T) {
	payload := "https://example.com | category=AI | title=Example"
	parsed := parsePipePayload(payload)

	if parsed["url"] != "https://example.com" {
		t.Fatalf("expected url to be parsed")
	}
	if parsed["category"] != "AI" {
		t.Fatalf("expected category to be parsed")
	}
	if parsed["title"] != "Example" {
		t.Fatalf("expected title to be parsed")
	}
}

func TestFirstNonEmpty(t *testing.T) {
	result := firstNonEmpty("", "   ", "ok")
	if result != "ok" {
		t.Fatalf("expected ok, got %q", result)
	}
}
