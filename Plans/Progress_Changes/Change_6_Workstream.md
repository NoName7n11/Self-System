# Change 6 Workstream - Content Extraction Pipeline

Date: 2026-06-03
Status: WS1 COMPLETE — WS2 COMPLETE — WS3 COMPLETE — WS4 COMPLETE — WS5 COMPLETE
Scope: Build the real ingestion pipeline for URLs, PDFs, and images so resources contain actual extracted content.

## Objective

Replace the current stub ingestion (metadata-only) with a real extraction pipeline that fetches and parses content from URLs, PDFs, and images. This is the prerequisite for every intelligence and lifecycle feature downstream.

## Guiding Constraints

- Extraction must be async — the resource is saved immediately, extraction runs in the background.
- Failures must be graceful — if extraction fails, the resource remains usable with whatever data it has.
- Respect the two-tier design: skim (fast, lightweight) fires first; deep extraction queues separately.
- No new infrastructure dependencies if avoidable — prefer pure Go libraries.
- All extraction paths must be tested with real fixture files (HTML, PDF, image).

## Workstream 1 — URL / Web Scraping (Skim Tier) — COMPLETE (Session 16)

Objective:
Extract title, description, and main body text from a URL in the skim pass (2–5 seconds budget).

Key tasks:
- [x] Add HTTP fetch with timeout, redirect limit, and User-Agent header.
- [x] Parse HTML: extract `<title>`, `<meta name="description">`, and main content (strip nav/footer/ads).
- [x] Detect page type heuristics: article, landing page, event page, product page.
- [x] Store extracted title + description into the resource record on skim completion.
- [ ] Emit a `ResourceSkimCompleted` event into the event store. — deferred to WS5 integration

Deliverables:
- [x] `internal/extractor/url_extractor.go` — fetch + parse logic.
- [x] `internal/extractor/url_extractor_test.go` — unit tests with HTML fixtures.
- [x] Skim hook wired into resource creation flow.

Done criteria:
- [x] Creating a resource with a URL triggers skim extraction and populates title + description.
- [x] Extraction timeout and failure are handled gracefully (resource saved with original input).

## Workstream 2 — PDF Parsing — COMPLETE (Session 17)

Objective:
Extract text content from uploaded PDF files, size-stratified by the spec.

Key tasks:
- [x] Integrate a pure Go PDF text extraction library (`github.com/ledongthuc/pdf`).
- [x] Small PDFs (< 5 pages / < 2MB): full text extraction.
- [x] Medium PDFs (5–50 pages): extract first 2 + last 2 pages.
- [x] Large PDFs (> 50 pages): extract first 2 pages only.
- [x] Store extracted text in resource `extracted_data` payload.
- [x] Emit `ResourcePDFExtracted` event type (reserved).

Deliverables:
- [x] `internal/extractor/pdf_extractor.go` — size-stratified extraction logic.
- [x] `internal/extractor/pdf_extractor_test.go` — tests with small/medium/large PDF fixtures via embedded builder.

Done criteria:
- [x] PDF resources have extracted text populated after processing.
- [x] Size thresholds are enforced and tested.

## Workstream 3 — Image Processing and OCR — COMPLETE (Session 18)

> **Note:** OCR via Tesseract was deferred because it requires a C system dependency
> incompatible with the pure-Go build. OCR is delivered in Change 7 via AI vision
> (the same AI manager already used for classification). Thumbnail generation and
> image type classification were delivered as planned.

Objective:
Classify image type and generate thumbnails. OCR deferred to Change 7 (AI vision layer).

Key tasks:
- [ ] Integrate a Go OCR library — deferred to Change 7 WS5 (AI vision call).
- [x] Generate a thumbnail for graph display (nearest-neighbour scale to 200×200, PNG output).
- [x] Classify image type: screenshot, diagram, photo, unknown (heuristic: format + aspect ratio + filename hint).
- [x] Store image type + thumbnail in `extracted_data`.
- [x] Emit `ResourceImageProcessed` event type (reserved).

Deliverables:
- [x] `internal/extractor/image_extractor.go` — classification + thumbnail logic (pure stdlib).
- [x] `internal/extractor/image_extractor_test.go` — 9 tests with programmatically generated fixtures.

Done criteria:
- [x] Image resources have image type and thumbnail populated after processing.
- [ ] OCR text populated — deferred to Change 7.
- [x] Thumbnail is generated and stored as base64 in `extracted_data`.

## Workstream 4 — Event Detection (URL / Document) — COMPLETE (Session 19)

Objective:
Detect actionable events in extracted content (hackathon deadlines, conference dates, internship applications) and auto-create reminders.

Key tasks:
- [x] Parse extracted text for event keywords (20 ordered keywords, multi-word phrases prioritised).
- [x] Extract dates using 5 regex patterns (ISO, full/short month name, day-month-year, US numeric).
- [x] Strip ordinal suffixes (1st, 2nd, 3rd, 4th → 1, 2, 3, 4) before parsing.
- [x] If event detected with future date: create a reminder record linked to the resource (wired in WS5).
- [x] Emit `ResourceEventDetected` event type (reserved).

Deliverables:
- [x] `internal/extractor/event_detector.go` — keyword + date extraction logic.
- [x] Integration with reminder service (wired via `runEventDetection` in WS5).
- [x] `internal/extractor/event_detector_test.go` — 13 tests covering all date formats, edge cases, and false-positive check.

Done criteria:
- [x] Resources with deadline/event content auto-generate linked reminders (via deep processor).
- [x] False positive rate is acceptable (no reminder for plain article text in test fixtures).

## Workstream 5 — Integration, Wiring, and CI Gate — COMPLETE (Session 20)

Objective:
Wire all extractors into the resource creation and deep processing flows, update the event store, and add CI coverage.

Key tasks:
- [x] Wire skim extractor into `ResourceService.Create` post-save (async goroutine).
- [x] Wire PDF/image extractors into the deep processing worker (`runExtractionForResource`).
- [x] Wire event detection + reminder creation into the deep processing worker (`runEventDetection`).
- [x] Update `extracted_data` field in resource via `UpdateExtractedData` after extraction completes.
- [x] Add integration tests: create resource → verify extracted fields populated.
- [x] Add CI gate: `go test ./internal/extractor/... ./test/integration/...`.

Deliverables:
- [x] `internal/extractor/fetcher.go` — raw content fetcher (30s timeout, 20 MiB cap).
- [x] Updated `internal/service/resource_service.go` with skim hook + `UpdateExtractedData`.
- [x] Updated `internal/service/deep_processor.go` with extraction dispatch + event detection.
- [x] `test/integration/extraction_integration_test.go` — 3 integration tests.
- [x] Makefile `extraction-test` target.
- [x] CI workflow extraction gate step in `.github/workflows/event-sourcing-gates.yml`.

Done criteria:
- [x] End-to-end: URL resource created → skim fires → title/description populated → deep queue → full content extracted.
- [x] All extractor packages have > 80% unit test coverage.
- [x] Full `go test ./...` passes.

## Planned Milestones

- [x] Milestone 6A: URL scraping skim tier live (WS1 complete).
- [x] Milestone 6B: PDF and image extraction complete (WS2 + WS3 complete).
- [x] Milestone 6C: Event detection and reminder auto-creation (WS4 complete).
- [x] Milestone 6D: Full integration, wiring, and CI gate (WS5 complete).

## Change 6 Definition of Done

- [x] URL, PDF, and image resources all produce real extracted content after creation.
- [x] Two-tier pipeline (skim fast, deep thorough) is functionally real, not a stub.
- [x] Event detection auto-creates reminders for deadline/event content.
- [x] All extractor packages are tested and gated in CI.
- [x] `go test ./...` passes with no regressions.
