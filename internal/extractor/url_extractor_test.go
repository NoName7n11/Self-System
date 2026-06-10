package extractor_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"selfsystems/internal/extractor"
)

func serveFixture(t *testing.T, filename, contentType string) *httptest.Server {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", filename))
	if err != nil {
		t.Fatalf("read fixture %s: %v", filename, err)
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", contentType)
		_, _ = w.Write(data)
	}))
}

func TestURLExtractor_ArticlePage(t *testing.T) {
	srv := serveFixture(t, "article.html", "text/html; charset=utf-8")
	defer srv.Close()

	ex := extractor.NewURLExtractor()
	result, err := ex.Extract(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// OG title preferred over plain <title>
	if result.Title != "Introduction to Machine Learning" {
		t.Errorf("title = %q, want OG title", result.Title)
	}
	// OG description preferred over plain meta description
	if result.Description != "Learn ML from scratch with practical examples." {
		t.Errorf("description = %q, want OG description", result.Description)
	}
	// Article signals detected
	if result.PageType != extractor.PageTypeArticle {
		t.Errorf("page_type = %q, want %q", result.PageType, extractor.PageTypeArticle)
	}
	// Nav / footer / script text must be absent
	if strings.Contains(result.MainText, "skip me") {
		t.Error("main text contains script content (should be stripped)")
	}
	if strings.Contains(result.MainText, "Copyright") {
		t.Error("main text contains footer content (should be stripped)")
	}
	// Body article text must be present
	if !strings.Contains(result.MainText, "Machine learning") {
		t.Error("main text missing article body content")
	}
}

func TestURLExtractor_EventPage(t *testing.T) {
	srv := serveFixture(t, "event.html", "text/html; charset=utf-8")
	defer srv.Close()

	ex := extractor.NewURLExtractor()
	result, err := ex.Extract(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Title != "AI Hackathon 2026" {
		t.Errorf("title = %q, want %q", result.Title, "AI Hackathon 2026")
	}
	if result.PageType != extractor.PageTypeEvent {
		t.Errorf("page_type = %q, want %q", result.PageType, extractor.PageTypeEvent)
	}
}

func TestURLExtractor_MinimalPage(t *testing.T) {
	srv := serveFixture(t, "minimal.html", "text/html; charset=utf-8")
	defer srv.Close()

	ex := extractor.NewURLExtractor()
	result, err := ex.Extract(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Title != "Minimal Page" {
		t.Errorf("title = %q, want %q", result.Title, "Minimal Page")
	}
	if result.PageType != extractor.PageTypeUnknown {
		t.Errorf("page_type = %q, want %q", result.PageType, extractor.PageTypeUnknown)
	}
}

func TestURLExtractor_NonHTMLContentType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write([]byte("%PDF-1.4"))
	}))
	defer srv.Close()

	ex := extractor.NewURLExtractor()
	_, err := ex.Extract(context.Background(), srv.URL)
	if err == nil {
		t.Error("expected error for non-HTML content type, got nil")
	}
}

func TestURLExtractor_NonSuccessStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	ex := extractor.NewURLExtractor()
	_, err := ex.Extract(context.Background(), srv.URL)
	if err == nil {
		t.Error("expected error for 404, got nil")
	}
}

func TestURLExtractor_TimeoutRespected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow server — context will cancel before this returns
		select {
		case <-r.Context().Done():
			return
		case <-time.After(30 * time.Second):
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte("<html><head><title>slow</title></head></html>"))
		}
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ex := extractor.NewURLExtractor()
	_, err := ex.Extract(ctx, srv.URL)
	if err == nil {
		t.Error("expected error when context times out, got nil")
	}
}

func TestURLExtractor_MetaDescriptionFallback(t *testing.T) {
	// No OG tags — should fall back to plain meta description
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!DOCTYPE html><html><head>
			<title>Plain Title</title>
			<meta name="description" content="Plain meta description.">
		</head><body><p>Body content here.</p></body></html>`))
	}))
	defer srv.Close()

	ex := extractor.NewURLExtractor()
	result, err := ex.Extract(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Title != "Plain Title" {
		t.Errorf("title = %q, want %q", result.Title, "Plain Title")
	}
	if result.Description != "Plain meta description." {
		t.Errorf("description = %q, want %q", result.Description, "Plain meta description.")
	}
}
