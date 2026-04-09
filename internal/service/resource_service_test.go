package service

import "testing"

func TestNormalizeURL(t *testing.T) {
	normalized, host, err := normalizeURL("https://www.example.com/path")
	if err != nil {
		t.Fatalf("expected valid URL, got error: %v", err)
	}
	if host != "example.com" {
		t.Fatalf("expected host example.com, got %q", host)
	}
	if normalized == "" {
		t.Fatalf("expected non-empty normalized URL")
	}
}

func TestNormalizeURLRejectsInvalid(t *testing.T) {
	_, _, err := normalizeURL("not-a-url")
	if err == nil {
		t.Fatalf("expected error for invalid url")
	}
}
