package extractor_test

import (
	"context"
	"testing"
	"time"

	"selfsystems/internal/extractor"
)

func TestEventDetector_DeadlineWithFullMonthDate(t *testing.T) {
	text := "Submit your application before the deadline: January 25, 2030."

	d := extractor.NewEventDetector()
	result := d.Detect(context.Background(), text)

	if !result.IsEvent {
		t.Fatal("expected IsEvent = true")
	}
	sig := findSignal(result, "deadline")
	if sig == nil {
		t.Fatal("expected signal for keyword 'deadline'")
	}
	if sig.DateText == "" {
		t.Error("expected DateText to be populated")
	}
	if sig.Date.IsZero() {
		t.Error("expected Date to be parsed")
	}
	if sig.Date.Year() != 2030 || sig.Date.Month() != time.January || sig.Date.Day() != 25 {
		t.Errorf("date = %v, want 2030-01-25", sig.Date)
	}
}

func TestEventDetector_ApplyByWithShortMonth(t *testing.T) {
	text := "Apply by Mar 31, 2030 to participate in the hackathon."

	d := extractor.NewEventDetector()
	result := d.Detect(context.Background(), text)

	if !result.IsEvent {
		t.Fatal("expected IsEvent = true")
	}
	sig := findSignal(result, "apply by")
	if sig == nil {
		t.Fatal("expected signal for keyword 'apply by'")
	}
	if sig.Date.Month() != time.March || sig.Date.Day() != 31 || sig.Date.Year() != 2030 {
		t.Errorf("date = %v, want 2030-03-31", sig.Date)
	}
}

func TestEventDetector_ISODateFormat(t *testing.T) {
	text := "Conference registration closes on 2030-06-15. Book your spot now."

	d := extractor.NewEventDetector()
	result := d.Detect(context.Background(), text)

	if !result.IsEvent {
		t.Fatal("expected IsEvent = true")
	}
	sig := findSignal(result, "registration closes")
	if sig == nil {
		t.Fatal("expected signal for keyword 'registration closes'")
	}
	if sig.Date.Year() != 2030 || sig.Date.Month() != time.June || sig.Date.Day() != 15 {
		t.Errorf("date = %v, want 2030-06-15", sig.Date)
	}
}

func TestEventDetector_USNumericDate(t *testing.T) {
	text := "Submit by 01/31/2030. Late submissions will not be accepted."

	d := extractor.NewEventDetector()
	result := d.Detect(context.Background(), text)

	if !result.IsEvent {
		t.Fatal("expected IsEvent = true")
	}
	sig := findSignal(result, "submit by")
	if sig == nil {
		t.Fatal("expected signal for keyword 'submit by'")
	}
	if sig.Date.Month() != time.January || sig.Date.Day() != 31 || sig.Date.Year() != 2030 {
		t.Errorf("date = %v, want 2030-01-31", sig.Date)
	}
}

func TestEventDetector_OrdinalDateStripped(t *testing.T) {
	// "January 25th, 2030" should parse despite ordinal suffix
	text := "Deadline: January 25th, 2030."

	d := extractor.NewEventDetector()
	result := d.Detect(context.Background(), text)

	if !result.IsEvent {
		t.Fatal("expected IsEvent = true")
	}
	sig := findSignal(result, "deadline")
	if sig == nil {
		t.Fatal("expected signal for keyword 'deadline'")
	}
	if sig.Date.Day() != 25 {
		t.Errorf("date day = %d, want 25 (ordinal suffix not stripped correctly)", sig.Date.Day())
	}
}

func TestEventDetector_DayMonthYearFormat(t *testing.T) {
	text := "Workshop registration is open until 15 March 2030."

	d := extractor.NewEventDetector()
	result := d.Detect(context.Background(), text)

	if !result.IsEvent {
		t.Fatal("expected IsEvent = true")
	}
	sig := findSignal(result, "workshop")
	if sig == nil {
		t.Fatal("expected signal for keyword 'workshop'")
	}
	if sig.Date.Month() != time.March || sig.Date.Day() != 15 || sig.Date.Year() != 2030 {
		t.Errorf("date = %v, want 2030-03-15", sig.Date)
	}
}

func TestEventDetector_KeywordWithoutDate(t *testing.T) {
	// Hackathon found but no parseable date nearby — signal still created, date is zero.
	text := "Join our annual hackathon and compete for great prizes!"

	d := extractor.NewEventDetector()
	result := d.Detect(context.Background(), text)

	if !result.IsEvent {
		t.Fatal("expected IsEvent = true (keyword found)")
	}
	sig := findSignal(result, "hackathon")
	if sig == nil {
		t.Fatal("expected signal for keyword 'hackathon'")
	}
	if !sig.Date.IsZero() {
		t.Errorf("expected zero date when no date in text, got %v", sig.Date)
	}
}

func TestEventDetector_NoEventKeywords(t *testing.T) {
	text := "Machine learning is transforming how companies build products. " +
		"Neural networks have shown impressive results across many domains."

	d := extractor.NewEventDetector()
	result := d.Detect(context.Background(), text)

	if result.IsEvent {
		t.Errorf("expected IsEvent = false for plain article, got true (signals: %+v)", result.Signals)
	}
	if len(result.Signals) != 0 {
		t.Errorf("expected 0 signals, got %d", len(result.Signals))
	}
}

func TestEventDetector_MultipleKeywords(t *testing.T) {
	text := "AI Hackathon 2030. Deadline to apply by March 1, 2030. " +
		"Registration closes on 2030-02-25. Early bird pricing available."

	d := extractor.NewEventDetector()
	result := d.Detect(context.Background(), text)

	if !result.IsEvent {
		t.Fatal("expected IsEvent = true")
	}
	// At least deadline, apply by, registration closes, early bird, hackathon
	if len(result.Signals) < 3 {
		t.Errorf("expected ≥ 3 signals, got %d", len(result.Signals))
	}
}

func TestEventDetector_SnippetPopulated(t *testing.T) {
	text := "You must register by January 10, 2030 to attend."

	d := extractor.NewEventDetector()
	result := d.Detect(context.Background(), text)

	if !result.IsEvent {
		t.Fatal("expected IsEvent = true")
	}
	sig := findSignal(result, "register by")
	if sig == nil {
		t.Fatal("expected signal for 'register by'")
	}
	if sig.Snippet == "" {
		t.Error("expected Snippet to be populated")
	}
	if len([]rune(sig.Snippet)) > 150 {
		t.Errorf("Snippet length %d exceeds 150 runes", len([]rune(sig.Snippet)))
	}
}

func TestEventDetector_EmptyText(t *testing.T) {
	d := extractor.NewEventDetector()
	result := d.Detect(context.Background(), "")
	if result.IsEvent {
		t.Error("expected IsEvent = false for empty text")
	}
}

func TestEventDetector_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	d := extractor.NewEventDetector()
	result := d.Detect(ctx, "deadline January 1, 2030")
	if result.IsEvent {
		t.Error("expected IsEvent = false for cancelled context")
	}
}

func TestEventDetector_HasFutureDate(t *testing.T) {
	// A signal with a far-future date should have HasFutureDate() == true.
	text := "Apply by January 1, 2099."
	d := extractor.NewEventDetector()
	result := d.Detect(context.Background(), text)

	if !result.IsEvent {
		t.Fatal("expected IsEvent = true")
	}
	sig := findSignal(result, "apply by")
	if sig == nil {
		t.Fatal("no signal found")
	}
	if !sig.HasFutureDate() {
		t.Errorf("expected HasFutureDate() = true for date %v", sig.Date)
	}
}

// ---- helpers ----------------------------------------------------------------

func findSignal(r extractor.EventDetectResult, keyword string) *extractor.EventSignal {
	for i := range r.Signals {
		if r.Signals[i].Keyword == keyword {
			return &r.Signals[i]
		}
	}
	return nil
}
