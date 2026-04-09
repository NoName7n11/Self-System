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

func TestParseLimit(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		fallback int
		want     int
	}{
		{name: "empty uses fallback", raw: "", fallback: 20, want: 20},
		{name: "invalid uses fallback", raw: "x", fallback: 20, want: 20},
		{name: "negative uses fallback", raw: "-10", fallback: 20, want: 20},
		{name: "valid value", raw: "15", fallback: 20, want: 15},
		{name: "clamped to max", raw: "120", fallback: 20, want: 100},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseLimit(tc.raw, tc.fallback)
			if got != tc.want {
				t.Fatalf("expected %d, got %d", tc.want, got)
			}
		})
	}
}

func TestParseGraphLimit(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		fallback int
		want     int
	}{
		{name: "empty uses fallback", raw: "", fallback: 1000, want: 1000},
		{name: "invalid uses fallback", raw: "x", fallback: 1000, want: 1000},
		{name: "negative uses fallback", raw: "-1", fallback: 1000, want: 1000},
		{name: "valid value", raw: "1200", fallback: 1000, want: 1200},
		{name: "clamped to max", raw: "10000", fallback: 1000, want: 5000},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseGraphLimit(tc.raw, tc.fallback)
			if got != tc.want {
				t.Fatalf("expected %d, got %d", tc.want, got)
			}
		})
	}
}

func TestParsePipePayloadQueryMode(t *testing.T) {
	payload := "ai graphs | query=knowledge graph | limit=12"
	parsed := parsePipePayload(payload)

	if parsed["value"] != "ai graphs" {
		t.Fatalf("expected value to be parsed")
	}
	if parsed["query"] != "knowledge graph" {
		t.Fatalf("expected query to be parsed")
	}
	if parsed["limit"] != "12" {
		t.Fatalf("expected limit to be parsed")
	}
}
