package extractor

import (
	"context"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// EventSignal represents a single detected event signal in a piece of text.
type EventSignal struct {
	Keyword  string    // the matched keyword that triggered detection
	Date     time.Time // extracted date (zero value if none found near the keyword)
	DateText string    // raw date string as it appeared in the text
	Snippet  string    // ≤ 150-char context window around the keyword
}

// HasFutureDate reports whether the signal contains a date that is in the future
// relative to now. Used by the reminder creation layer (WS5) to skip past events.
func (s EventSignal) HasFutureDate() bool {
	return !s.Date.IsZero() && s.Date.After(time.Now())
}

// EventDetectResult is the output of EventDetector.Detect.
type EventDetectResult struct {
	IsEvent bool          // true when at least one event keyword was found
	Signals []EventSignal // one entry per unique keyword matched
}

// EventDetector scans extracted text for actionable event signals and nearby dates.
// It is pure text analysis — no network calls, no side effects, safe for concurrent use.
// Reminder creation is handled upstream (Change 6 WS5 wiring).
type EventDetector struct{}

// NewEventDetector returns an EventDetector ready to use.
func NewEventDetector() *EventDetector { return &EventDetector{} }

// Detect scans text for event keywords and nearby date expressions.
// Each unique keyword produces at most one EventSignal.
// IsEvent is true when any keyword is found; callers should check
// EventSignal.HasFutureDate before creating reminders to skip past events.
func (d *EventDetector) Detect(ctx context.Context, text string) EventDetectResult {
	if ctx.Err() != nil || strings.TrimSpace(text) == "" {
		return EventDetectResult{}
	}

	lower := strings.ToLower(text)
	seen := make(map[string]bool)
	var signals []EventSignal

	for _, kw := range orderedKeywords {
		if seen[kw] {
			continue
		}
		idx := strings.Index(lower, kw)
		if idx < 0 {
			continue
		}
		seen[kw] = true

		// Search for a date in a ±250-char window around the keyword.
		winStart := max0(idx - 250)
		winEnd := minInt(len(text), idx+len(kw)+250)
		window := text[winStart:winEnd]

		date, dateText := findNearestDate(window)

		// Build a compact snippet for display (≤ 150 chars).
		snipStart := max0(idx - 60)
		snipEnd := minInt(len(text), idx+len(kw)+60)
		snippet := strings.TrimSpace(text[snipStart:snipEnd])
		if len([]rune(snippet)) > 150 {
			snippet = string([]rune(snippet)[:150])
		}

		signals = append(signals, EventSignal{
			Keyword:  kw,
			Date:     date,
			DateText: dateText,
			Snippet:  snippet,
		})
	}

	return EventDetectResult{
		IsEvent: len(signals) > 0,
		Signals: signals,
	}
}

// ---- Keyword list -----------------------------------------------------------

// orderedKeywords lists all event signals in priority order (strongest first).
// Multi-word phrases must appear before their component words to avoid
// partial matches ("apply by" before "apply").
var orderedKeywords = []string{
	"applications close",
	"registration closes",
	"last day to apply",
	"applications due",
	"apply by",
	"register by",
	"sign up by",
	"submit by",
	"due date",
	"due by",
	"deadline",
	"early bird",
	"limited spots",
	"apply now",
	"applications open",
	"hackathon",
	"conference",
	"workshop",
	"competition",
	"internship",
}

// ---- Date extraction --------------------------------------------------------

type datePattern struct {
	re      *regexp.Regexp
	formats []string // time.Parse format strings to try in order
}

// ordinalSuffix strips English ordinal suffixes (1st, 2nd, 3rd, 4th → 1, 2, 3, 4).
var ordinalRe = regexp.MustCompile(`\b(\d{1,2})(st|nd|rd|th)\b`)

var datePatterns = []datePattern{
	// ISO: 2026-01-25 or 2026/01/25
	{
		re:      regexp.MustCompile(`\b(\d{4}[-/]\d{2}[-/]\d{2})\b`),
		formats: []string{"2006-01-02", "2006/01/02"},
	},
	// Full month name: "January 25, 2026" / "January 25 2026"
	{
		re: regexp.MustCompile(`(?i)\b((?:january|february|march|april|may|june|july|august|september|october|november|december)\s+\d{1,2}(?:st|nd|rd|th)?,?\s+\d{4})\b`),
		formats: []string{
			"January 2, 2006", "January 2 2006",
			"January 02, 2006", "January 02 2006",
		},
	},
	// Short month name: "Jan 25, 2026" / "Jan. 25, 2026"
	{
		re: regexp.MustCompile(`(?i)\b((?:jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)\.?\s+\d{1,2}(?:st|nd|rd|th)?,?\s+\d{4})\b`),
		formats: []string{
			"Jan 2, 2006", "Jan. 2, 2006", "Jan 2 2006",
			"Jan 02, 2006", "Jan. 02, 2006",
		},
	},
	// Day Month Year: "25 January 2026"
	{
		re:      regexp.MustCompile(`(?i)\b(\d{1,2}\s+(?:january|february|march|april|may|june|july|august|september|october|november|december)\s+\d{4})\b`),
		formats: []string{"2 January 2006", "02 January 2006"},
	},
	// US numeric: 01/25/2026 or 1/25/2026
	{
		re:      regexp.MustCompile(`\b(\d{1,2}/\d{1,2}/\d{4})\b`),
		formats: []string{"1/2/2006", "01/02/2006"},
	},
}

// findNearestDate scans text for the first recognisable date and returns the
// parsed time and the raw matched string. Returns zero time and empty string
// if no date is found.
func findNearestDate(text string) (time.Time, string) {
	// Strip ordinal suffixes so "January 25th, 2026" → "January 25, 2026"
	cleaned := ordinalRe.ReplaceAllString(text, "$1")

	type candidate struct {
		idx int
		raw string
		t   time.Time
	}
	var best *candidate

	for _, pat := range datePatterns {
		loc := pat.re.FindStringIndex(cleaned)
		if loc == nil {
			continue
		}
		raw := cleaned[loc[0]:loc[1]]

		t := tryParseDate(raw, pat.formats)
		if t.IsZero() {
			continue
		}

		if best == nil || loc[0] < best.idx {
			c := candidate{idx: loc[0], raw: raw, t: t}
			best = &c
		}
	}

	if best == nil {
		return time.Time{}, ""
	}
	return best.t.UTC(), best.raw
}

// tryParseDate tries each format string in order and returns the first success.
func tryParseDate(raw string, formats []string) time.Time {
	// Normalise: collapse internal whitespace, trim.
	norm := strings.Join(strings.FieldsFunc(raw, unicode.IsSpace), " ")
	norm = strings.TrimSpace(norm)

	for _, f := range formats {
		t, err := time.Parse(f, norm)
		if err == nil {
			return t
		}
	}
	return time.Time{}
}

// ---- Helpers ----------------------------------------------------------------

func max0(n int) int {
	if n < 0 {
		return 0
	}
	return n
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
