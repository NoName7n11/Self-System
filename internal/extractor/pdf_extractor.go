package extractor

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/ledongthuc/pdf"
)

const (
	pdfSmallMaxPages  = 5
	pdfSmallMaxBytes  = 2 << 20 // 2 MiB
	pdfMediumMaxPages = 50

	pdfMaxExtractRunes = 20000 // upper bound on extracted text kept in memory
)

// PDFSizeClass describes how much text the extractor pulls from a PDF.
const (
	PDFSizeClassSmall  = "small"  // < 5 pages / < 2 MiB — full text
	PDFSizeClassMedium = "medium" // 5–50 pages — first 2 + last 2 pages
	PDFSizeClassLarge  = "large"  // > 50 pages — first 2 pages only
)

// PDFExtractResult holds the data produced by extracting a PDF file.
type PDFExtractResult struct {
	Text      string
	PageCount int
	SizeClass string
}

// PDFExtractor extracts text from PDF byte content. It is safe for concurrent use.
type PDFExtractor struct{}

// NewPDFExtractor returns a PDFExtractor ready to use.
func NewPDFExtractor() *PDFExtractor { return &PDFExtractor{} }

// Extract parses content as a PDF and returns extracted text.
// The amount of text extracted depends on the page count and file size:
//   - small (< 5 pages / < 2 MiB) → full text
//   - medium (5–50 pages)         → first 2 + last 2 pages
//   - large (> 50 pages)          → first 2 pages
//
// A non-nil error means the bytes could not be parsed as a valid PDF.
func (e *PDFExtractor) Extract(ctx context.Context, content []byte) (PDFExtractResult, error) {
	if err := ctx.Err(); err != nil {
		return PDFExtractResult{}, err
	}
	if len(content) == 0 {
		return PDFExtractResult{}, fmt.Errorf("pdf content is empty")
	}

	r, err := pdf.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return PDFExtractResult{}, fmt.Errorf("parse pdf: %w", err)
	}

	numPages := r.NumPage()
	if numPages < 0 {
		numPages = 0
	}
	sizeClass := classifyPDF(numPages, int64(len(content)))

	var text string
	switch sizeClass {
	case PDFSizeClassSmall:
		text, err = extractAllPDFPages(r, numPages)
	case PDFSizeClassMedium:
		text, err = extractMediumPDFPages(r, numPages)
	default: // large
		text, err = extractSelectedPDFPages(r, []int{1, 2})
	}
	if err != nil {
		return PDFExtractResult{}, err
	}

	return PDFExtractResult{
		Text:      truncatePDF(text, pdfMaxExtractRunes),
		PageCount: numPages,
		SizeClass: sizeClass,
	}, nil
}

// classifyPDF returns the size class based on page count and file size.
func classifyPDF(numPages int, sizeBytes int64) string {
	if numPages < pdfSmallMaxPages && sizeBytes < pdfSmallMaxBytes {
		return PDFSizeClassSmall
	}
	if numPages <= pdfMediumMaxPages {
		return PDFSizeClassMedium
	}
	return PDFSizeClassLarge
}

// extractAllPDFPages reads all pages using the library's optimised reader.
func extractAllPDFPages(r *pdf.Reader, numPages int) (string, error) {
	if numPages == 0 {
		return "", nil
	}
	ir, err := r.GetPlainText()
	if err != nil {
		return "", fmt.Errorf("read pdf text: %w", err)
	}
	var buf bytes.Buffer
	_, err = buf.ReadFrom(ir)
	return buf.String(), err
}

// extractMediumPDFPages reads the first 2 and last 2 pages (de-duped).
func extractMediumPDFPages(r *pdf.Reader, numPages int) (string, error) {
	want := []int{1, 2, numPages - 1, numPages}
	seen := map[int]bool{}
	unique := make([]int, 0, 4)
	for _, p := range want {
		if p >= 1 && p <= numPages && !seen[p] {
			seen[p] = true
			unique = append(unique, p)
		}
	}
	return extractSelectedPDFPages(r, unique)
}

// extractSelectedPDFPages extracts text from the given 1-indexed page numbers.
// Pages that fail to parse are skipped rather than failing the whole extraction.
func extractSelectedPDFPages(r *pdf.Reader, pageNums []int) (string, error) {
	fonts := make(map[string]*pdf.Font)
	var sb strings.Builder
	for _, n := range pageNums {
		if n < 1 || n > r.NumPage() {
			continue
		}
		page := r.Page(n)
		// Warm the font cache for this page.
		for _, name := range page.Fonts() {
			if _, ok := fonts[name]; !ok {
				f := page.Font(name)
				fonts[name] = &f
			}
		}
		text, err := page.GetPlainText(fonts)
		if err != nil || strings.TrimSpace(text) == "" {
			continue
		}
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(text)
	}
	return sb.String(), nil
}

func truncatePDF(s string, maxRunes int) string {
	if len([]rune(s)) <= maxRunes {
		return s
	}
	return string([]rune(s)[:maxRunes])
}
