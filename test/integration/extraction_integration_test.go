package integration_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"selfsystems/internal/ai"
	"selfsystems/internal/domain"
	"selfsystems/internal/extractor"
	sqliterepo "selfsystems/internal/repository/sqlite"
	"selfsystems/internal/service"
)

// ---- shared test setup ------------------------------------------------------

type extractionFixture struct {
	catSvc      *service.CategoryService
	resourceSvc *service.ResourceService
	reminderSvc *service.ReminderService
}

func newExtractionFixture(t *testing.T) extractionFixture {
	t.Helper()

	db, err := sqliterepo.Open(filepath.Join(t.TempDir(), "extract_test.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	catRepo := sqliterepo.NewCategoryRepository(db)
	resourceRepo := sqliterepo.NewResourceRepository(db)
	reminderRepo := sqliterepo.NewReminderRepository(db)

	aiMgr := ai.NewManager("heuristic")
	aiMgr.Register(ai.NewHeuristicProvider())
	aiMgr.SetFallback(ai.NewHeuristicProvider().Name())

	catSvc := service.NewCategoryService(catRepo)
	classifier := service.NewCategoryClassifier(catRepo, aiMgr)
	resourceSvc := service.NewResourceService(resourceRepo, catRepo, classifier, catSvc)
	reminderSvc := service.NewReminderService(reminderRepo)

	return extractionFixture{
		catSvc:      catSvc,
		resourceSvc: resourceSvc,
		reminderSvc: reminderSvc,
	}
}

func (f extractionFixture) newDeepProc(t *testing.T, opts ...func(*service.DeepProcessor) *service.DeepProcessor) *service.DeepProcessor {
	t.Helper()

	db, err := sqliterepo.Open(filepath.Join(t.TempDir(), "dp_test.db"))
	if err != nil {
		t.Fatalf("open deep proc db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	catRepo := sqliterepo.NewCategoryRepository(db)
	aiMgr := ai.NewManager("heuristic")
	aiMgr.Register(ai.NewHeuristicProvider())
	aiMgr.SetFallback(ai.NewHeuristicProvider().Name())
	catSvc := service.NewCategoryService(catRepo)

	p := service.NewDeepProcessor(
		f.resourceSvc, catRepo, catSvc, aiMgr,
		service.DeepProcessingSettings{
			MaxTasksPerMinute: 0,
			MaxTokensPerDay:   0,
		},
	)
	for _, opt := range opts {
		p = opt(p)
	}
	return p
}

// ---- tests ------------------------------------------------------------------

// TestDeepProcessor_EventDetection_CreatesReminder verifies the full path:
// resource with event content in extracted_data → ProcessDirect → reminder linked
// to the resource is auto-created with the extracted future date.
func TestDeepProcessor_EventDetection_CreatesReminder(t *testing.T) {
	fix := newExtractionFixture(t)
	ctx := context.Background()

	cat, err := fix.catSvc.EnsureByName(ctx, "Tech", domain.CategorySourceManual)
	if err != nil {
		t.Fatalf("ensure category: %v", err)
	}

	resource, err := fix.resourceSvc.Create(ctx, service.CreateResourceInput{
		URL:          "https://example.com/hackathon",
		Title:        "AI Hackathon 2099",
		CategoryName: cat.Name,
	})
	if err != nil {
		t.Fatalf("create resource: %v", err)
	}

	// Write event-bearing text into extracted_data.MainText.
	resource.ExtractedData.MainText = "Join our hackathon! Apply by January 1, 2099 to win prizes."
	if err := fix.resourceSvc.UpdateExtractedData(ctx, resource.ID, resource.ExtractedData); err != nil {
		t.Fatalf("update extracted_data: %v", err)
	}

	deepProc := fix.newDeepProc(t,
		func(p *service.DeepProcessor) *service.DeepProcessor {
			return p.
				WithEventDetector(extractor.NewEventDetector()).
				WithReminderService(fix.reminderSvc)
		},
	)

	if err := deepProc.ProcessDirect(ctx, resource.ID); err != nil {
		t.Fatalf("ProcessDirect: %v", err)
	}

	reminders, err := fix.reminderSvc.List(ctx, 20, 0)
	if err != nil {
		t.Fatalf("list reminders: %v", err)
	}

	var linked *domain.Reminder
	for i := range reminders {
		if reminders[i].ResourceID != nil && *reminders[i].ResourceID == resource.ID {
			linked = &reminders[i]
			break
		}
	}
	if linked == nil {
		t.Fatalf("no reminder linked to resource %q; reminders in DB: %d", resource.ID, len(reminders))
	}
	if linked.RemindAt.Year() != 2099 {
		t.Errorf("reminder RemindAt year = %d, want 2099", linked.RemindAt.Year())
	}
}

// TestDeepProcessor_NoEvent_NoReminder verifies that plain article text does
// not cause any reminder to be created.
func TestDeepProcessor_NoEvent_NoReminder(t *testing.T) {
	fix := newExtractionFixture(t)
	ctx := context.Background()

	cat, err := fix.catSvc.EnsureByName(ctx, "Tech", domain.CategorySourceManual)
	if err != nil {
		t.Fatalf("ensure category: %v", err)
	}
	resource, err := fix.resourceSvc.Create(ctx, service.CreateResourceInput{
		URL:          "https://example.com/article",
		Title:        "Introduction to Neural Networks",
		CategoryName: cat.Name,
	})
	if err != nil {
		t.Fatalf("create resource: %v", err)
	}

	resource.ExtractedData.MainText = "Neural networks are computational models inspired by biological neurons. " +
		"They are widely used in image recognition and natural language processing."
	if err := fix.resourceSvc.UpdateExtractedData(ctx, resource.ID, resource.ExtractedData); err != nil {
		t.Fatalf("update extracted_data: %v", err)
	}

	deepProc := fix.newDeepProc(t,
		func(p *service.DeepProcessor) *service.DeepProcessor {
			return p.
				WithEventDetector(extractor.NewEventDetector()).
				WithReminderService(fix.reminderSvc)
		},
	)

	if err := deepProc.ProcessDirect(ctx, resource.ID); err != nil {
		t.Fatalf("ProcessDirect: %v", err)
	}

	reminders, err := fix.reminderSvc.List(ctx, 20, 0)
	if err != nil {
		t.Fatalf("list reminders: %v", err)
	}
	for _, r := range reminders {
		if r.ResourceID != nil && *r.ResourceID == resource.ID {
			t.Errorf("unexpected reminder created for plain article: %+v", r)
		}
	}
}

// TestDeepProcessor_PDFExtraction verifies that a resource with a .pdf URL
// has its PDF text extracted and persisted in extracted_data after ProcessDirect.
func TestDeepProcessor_PDFExtraction(t *testing.T) {
	fix := newExtractionFixture(t)
	ctx := context.Background()

	// Serve a minimal valid PDF from an httptest server.
	pdfBytes := integrationTestPDF([]string{"PDF extraction test content", "Second page text"})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write(pdfBytes)
	}))
	defer srv.Close()

	deepProc := fix.newDeepProc(t,
		func(p *service.DeepProcessor) *service.DeepProcessor {
			return p.
				WithContentFetcher(extractor.NewContentFetcher()).
				WithPDFExtractor(extractor.NewPDFExtractor())
		},
	)

	cat, err := fix.catSvc.EnsureByName(ctx, "Docs", domain.CategorySourceManual)
	if err != nil {
		t.Fatalf("ensure category: %v", err)
	}

	// URL path ends in .pdf so inferSourceType returns "pdf".
	resource, err := fix.resourceSvc.Create(ctx, service.CreateResourceInput{
		URL:          srv.URL + "/document.pdf",
		Title:        "Test PDF Document",
		CategoryName: cat.Name,
	})
	if err != nil {
		t.Fatalf("create resource: %v", err)
	}

	if err := deepProc.ProcessDirect(ctx, resource.ID); err != nil {
		t.Fatalf("ProcessDirect: %v", err)
	}

	updated, err := fix.resourceSvc.GetByID(ctx, resource.ID)
	if err != nil || updated == nil {
		t.Fatalf("get updated resource: %v", err)
	}
	if updated.ExtractedData.PDFPageCount == 0 {
		t.Error("expected PDFPageCount > 0 after PDF extraction")
	}
	if !strings.Contains(updated.ExtractedData.PDFText, "PDF extraction test content") {
		t.Errorf("PDFText missing expected content; got: %q", updated.ExtractedData.PDFText)
	}
}

// ---- PDF fixture builder ----------------------------------------------------

// integrationTestPDF builds a minimal valid PDF for use in integration tests.
// Duplicated from extractor_test to avoid importing test packages across packages.
func integrationTestPDF(pageTexts []string) []byte {
	n := len(pageTexts)
	if n == 0 {
		n = 1
		pageTexts = []string{" "}
	}

	fontID := 3 + 2*n
	totalObjs := fontID
	offsets := make([]int, totalObjs+1) // 1-indexed

	var buf strings.Builder
	pos := func() int { return len(buf.String()) }

	startObj := func(id int) {
		offsets[id] = pos()
		buf.WriteString(fmt.Sprintf("%d 0 obj\n", id))
	}
	endObj := func() { buf.WriteString("endobj\n") }

	buf.WriteString("%PDF-1.4\n")

	var kids []string
	for i := 0; i < n; i++ {
		kids = append(kids, fmt.Sprintf("%d 0 R", 3+i))
	}

	startObj(1)
	buf.WriteString("<</Type /Catalog /Pages 2 0 R>>\n")
	endObj()

	startObj(2)
	buf.WriteString(fmt.Sprintf("<</Type /Pages /Kids [%s] /Count %d>>\n", strings.Join(kids, " "), n))
	endObj()

	for i := 0; i < n; i++ {
		contentID := 3 + n + i
		startObj(3 + i)
		buf.WriteString(fmt.Sprintf(
			"<</Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents %d 0 R /Resources <</Font <</F1 %d 0 R>>>>>>\n",
			contentID, fontID))
		endObj()
	}

	for i := 0; i < n; i++ {
		escaped := strings.ReplaceAll(pageTexts[i], "(", "\\(")
		escaped = strings.ReplaceAll(escaped, ")", "\\)")
		stream := fmt.Sprintf("BT /F1 12 Tf 72 720 Td (%s) Tj ET", escaped)
		startObj(3 + n + i)
		buf.WriteString(fmt.Sprintf("<</Length %d>>\nstream\n%s\nendstream\n", len(stream)+1, stream))
		endObj()
	}

	startObj(fontID)
	buf.WriteString("<</Type /Font /Subtype /Type1 /BaseFont /Helvetica>>\n")
	endObj()

	xrefOffset := pos()
	buf.WriteString(fmt.Sprintf("xref\n0 %d\n", totalObjs+1))
	buf.WriteString("0000000000 65535 f \n")
	for id := 1; id <= totalObjs; id++ {
		buf.WriteString(fmt.Sprintf("%010d 00000 n \n", offsets[id]))
	}
	buf.WriteString(fmt.Sprintf("trailer\n<</Size %d /Root 1 0 R>>\n", totalObjs+1))
	buf.WriteString(fmt.Sprintf("startxref\n%d\n%%%%EOF\n", xrefOffset))

	return []byte(buf.String())
}
