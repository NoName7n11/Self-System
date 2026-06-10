package extractor_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"selfsystems/internal/extractor"
)

// ---------------------------------------------------------------------------
// Minimal PDF builder for tests
// ---------------------------------------------------------------------------

// newTestPDF builds a syntactically valid minimal PDF with len(pageTexts) pages.
// Each element of pageTexts becomes the plain text body of that page, rendered
// with Helvetica 12pt at (72, 720). The resulting bytes can be fed directly to
// PDFExtractor.Extract.
func newTestPDF(pageTexts []string) []byte {
	n := len(pageTexts)
	if n == 0 {
		n = 1
		pageTexts = []string{" "}
	}

	// Object ID layout:
	//   1       — Catalog
	//   2       — Pages
	//   3..2+n  — Page objects
	//   3+n..2+2n — Content streams
	//   3+2n    — Font (Helvetica)
	fontID := 3 + 2*n
	totalObjs := fontID

	offsets := make([]int, totalObjs+1) // 1-indexed; offsets[i] = byte offset of obj i

	var buf strings.Builder
	pos := func() int { return len(buf.String()) }

	wf := func(format string, args ...any) { fmt.Fprintf(&buf, format, args...) }

	startObj := func(id int) {
		offsets[id] = pos()
		wf("%d 0 obj\n", id)
	}
	endObj := func() { wf("endobj\n") }

	wf("%%PDF-1.4\n")

	// Build kids list for Pages
	var kids []string
	for i := 0; i < n; i++ {
		kids = append(kids, fmt.Sprintf("%d 0 R", 3+i))
	}

	// Obj 1: Catalog
	startObj(1)
	wf("<</Type /Catalog /Pages 2 0 R>>\n")
	endObj()

	// Obj 2: Pages
	startObj(2)
	wf("<</Type /Pages /Kids [%s] /Count %d>>\n", strings.Join(kids, " "), n)
	endObj()

	// Page objects: 3 .. 2+n
	for i := 0; i < n; i++ {
		contentID := 3 + n + i
		startObj(3 + i)
		wf("<</Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents %d 0 R /Resources <</Font <</F1 %d 0 R>>>>>>\n",
			contentID, fontID)
		endObj()
	}

	// Content streams: 3+n .. 2+2n
	for i := 0; i < n; i++ {
		// Escape parentheses in the text string (PDF string literal)
		escaped := strings.ReplaceAll(pageTexts[i], "(", "\\(")
		escaped = strings.ReplaceAll(escaped, ")", "\\)")
		stream := fmt.Sprintf("BT /F1 12 Tf 72 720 Td (%s) Tj ET", escaped)
		// Length = len(stream) + 1 for the trailing \n before endstream
		startObj(3 + n + i)
		wf("<</Length %d>>\nstream\n%s\nendstream\n", len(stream)+1, stream)
		endObj()
	}

	// Font object
	startObj(fontID)
	wf("<</Type /Font /Subtype /Type1 /BaseFont /Helvetica>>\n")
	endObj()

	// XRef table — each entry must be exactly 20 bytes: "nnnnnnnnnn ggggg x \n"
	xrefOffset := pos()
	wf("xref\n0 %d\n", totalObjs+1)
	wf("0000000000 65535 f \n") // free object 0 (20 bytes including trailing space + LF)
	for id := 1; id <= totalObjs; id++ {
		wf("%010d 00000 n \n", offsets[id]) // 20 bytes each
	}

	wf("trailer\n<</Size %d /Root 1 0 R>>\n", totalObjs+1)
	wf("startxref\n%d\n%%%%EOF\n", xrefOffset)

	return []byte(buf.String())
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestPDFExtractor_SmallPDF_FullText(t *testing.T) {
	// 2 pages → small class → full text extracted
	content := newTestPDF([]string{"Page one content", "Page two content"})

	ex := extractor.NewPDFExtractor()
	result, err := ex.Extract(context.Background(), content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.SizeClass != extractor.PDFSizeClassSmall {
		t.Errorf("size_class = %q, want %q", result.SizeClass, extractor.PDFSizeClassSmall)
	}
	if result.PageCount != 2 {
		t.Errorf("page_count = %d, want 2", result.PageCount)
	}
	// Both pages should appear in the output
	if !strings.Contains(result.Text, "Page one content") {
		t.Errorf("text missing page 1 content: %q", result.Text)
	}
	if !strings.Contains(result.Text, "Page two content") {
		t.Errorf("text missing page 2 content: %q", result.Text)
	}
}

func TestPDFExtractor_MediumPDF_FirstAndLastPages(t *testing.T) {
	// 6 pages → medium class → pages 1, 2, 5, 6 extracted; 3, 4 skipped
	texts := []string{
		"First page alpha",
		"Second page beta",
		"Third page skip",
		"Fourth page skip",
		"Fifth page gamma",
		"Sixth page delta",
	}
	content := newTestPDF(texts)

	ex := extractor.NewPDFExtractor()
	result, err := ex.Extract(context.Background(), content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.SizeClass != extractor.PDFSizeClassMedium {
		t.Errorf("size_class = %q, want %q", result.SizeClass, extractor.PDFSizeClassMedium)
	}
	if result.PageCount != 6 {
		t.Errorf("page_count = %d, want 6", result.PageCount)
	}

	if !strings.Contains(result.Text, "First page alpha") {
		t.Error("text missing page 1")
	}
	if !strings.Contains(result.Text, "Second page beta") {
		t.Error("text missing page 2")
	}
	if !strings.Contains(result.Text, "Fifth page gamma") {
		t.Error("text missing page 5")
	}
	if !strings.Contains(result.Text, "Sixth page delta") {
		t.Error("text missing page 6")
	}
}

func TestPDFExtractor_LargePDF_FirstTwoPages(t *testing.T) {
	// 51 pages → large class → only pages 1 and 2 extracted
	texts := make([]string, 51)
	for i := range texts {
		texts[i] = fmt.Sprintf("Page %d unique text", i+1)
	}
	content := newTestPDF(texts)

	ex := extractor.NewPDFExtractor()
	result, err := ex.Extract(context.Background(), content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.SizeClass != extractor.PDFSizeClassLarge {
		t.Errorf("size_class = %q, want %q", result.SizeClass, extractor.PDFSizeClassLarge)
	}
	if result.PageCount != 51 {
		t.Errorf("page_count = %d, want 51", result.PageCount)
	}

	if !strings.Contains(result.Text, "Page 1 unique text") {
		t.Error("text missing page 1")
	}
	if !strings.Contains(result.Text, "Page 2 unique text") {
		t.Error("text missing page 2")
	}
	// Page 3 should NOT be present
	if strings.Contains(result.Text, "Page 3 unique text") {
		t.Error("text contains page 3 — large PDF should only extract pages 1–2")
	}
}

func TestPDFExtractor_EmptyContent(t *testing.T) {
	ex := extractor.NewPDFExtractor()
	_, err := ex.Extract(context.Background(), []byte{})
	if err == nil {
		t.Error("expected error for empty content, got nil")
	}
}

func TestPDFExtractor_InvalidBytes(t *testing.T) {
	ex := extractor.NewPDFExtractor()
	_, err := ex.Extract(context.Background(), []byte("this is not a pdf"))
	if err == nil {
		t.Error("expected error for invalid PDF bytes, got nil")
	}
}

func TestPDFExtractor_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled

	content := newTestPDF([]string{"some content"})
	ex := extractor.NewPDFExtractor()
	_, err := ex.Extract(ctx, content)
	if err == nil {
		t.Error("expected error for cancelled context, got nil")
	}
}

func TestPDFClassify_Thresholds(t *testing.T) {
	// Verify classification boundary: 4 pages, small file → small
	content := newTestPDF([]string{"a", "b", "c", "d"})
	ex := extractor.NewPDFExtractor()
	result, err := ex.Extract(context.Background(), content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SizeClass != extractor.PDFSizeClassSmall {
		t.Errorf("4-page PDF size_class = %q, want small", result.SizeClass)
	}

	// 5 pages → medium (regardless of file size for these small test PDFs)
	content5 := newTestPDF([]string{"a", "b", "c", "d", "e"})
	result5, err := ex.Extract(context.Background(), content5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result5.SizeClass != extractor.PDFSizeClassMedium {
		t.Errorf("5-page PDF size_class = %q, want medium", result5.SizeClass)
	}
}
