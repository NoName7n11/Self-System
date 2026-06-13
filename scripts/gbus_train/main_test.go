package main

import (
	"testing"
	"time"

	"selfsystems/internal/gbus"
)

// ev builds a signal payload at a deterministic, strictly-increasing time so
// chronological ordering in tests is unambiguous.
func ev(order int, signalType, categoryID string) gbus.GBUSSignalPayload {
	return gbus.GBUSSignalPayload{
		SignalType: signalType,
		CategoryID: categoryID,
		Weight:     gbus.SignalWeights[signalType],
		OccurredAt: time.Date(2026, 1, 1, 0, 0, order, 0, time.UTC),
	}
}

func TestBaselineWeights_PlainSumOrdering(t *testing.T) {
	events := []gbus.GBUSSignalPayload{
		ev(1, gbus.SignalManualClassification, "a"), // 1.0
		ev(2, gbus.SignalResourceSaved, "b"),        // 0.3
		ev(3, gbus.SignalResourceSaved, "b"),        // +0.3 = 0.6
	}
	w := baselineWeights(events)
	// "a" has the larger plain sum (1.0 vs 0.6) → normalises to 1.0.
	if w["a"] <= w["b"] {
		t.Fatalf("expected a > b, got a=%.3f b=%.3f", w["a"], w["b"])
	}
	if w["a"] != 1.0 {
		t.Errorf("max category should normalise to 1.0, got %.3f", w["a"])
	}
}

// A category backed only by passive resource_saved signals carries no explicit
// intent, so its confidence is 0 and the GBUS ranker zeroes it — while the
// plain baseline still scores it. This is the WS6 confidence discount in action.
func TestGBUSWeights_ConfidenceDiscountsPassiveOnlyCategory(t *testing.T) {
	events := []gbus.GBUSSignalPayload{
		ev(1, gbus.SignalManualClassification, "explicit"), // weight 1.0 ≥ threshold → evidence
		ev(2, gbus.SignalResourceSaved, "passive"),         // weight 0.3 < threshold → no evidence
		ev(3, gbus.SignalResourceSaved, "passive"),
		ev(4, gbus.SignalResourceSaved, "passive"),
	}
	gw := gbusWeights(events)
	if gw["passive"] != 0 {
		t.Errorf("passive-only category should be confidence-discounted to 0, got %.3f", gw["passive"])
	}
	if gw["explicit"] <= 0 {
		t.Errorf("explicit category should have positive weight, got %.3f", gw["explicit"])
	}

	// Baseline does NOT discount: passive still scores.
	bw := baselineWeights(events)
	if bw["passive"] <= 0 {
		t.Errorf("baseline should still score passive category, got %.3f", bw["passive"])
	}
}

func TestHitAtK_TiesAndRanking(t *testing.T) {
	candidates := map[string]struct{}{"a": {}, "b": {}, "c": {}}
	weights := map[string]float64{"a": 1.0, "b": 0.5, "c": 0.5}

	if !hitAtK("a", weights, candidates, 1) {
		t.Error("a is strict max, should hit@1")
	}
	if hitAtK("b", weights, candidates, 1) {
		t.Error("b is beaten by a, should miss@1")
	}
	// b and c tie at 0.5; only a (1.0) strictly beats them → both are top-3,
	// and each has exactly 1 category strictly better → hit@2.
	if !hitAtK("b", weights, candidates, 2) {
		t.Error("b should hit@2 (only a strictly better)")
	}
	if !hitAtK("c", weights, candidates, 3) {
		t.Error("c should hit@3")
	}

	// Unseen / zero-weight true category loses to any positive.
	if hitAtK("unseen", weights, candidates, 1) {
		t.Error("zero-weight category should not be top-1 when others are positive")
	}
}

// evaluate is prequential: each labeled instance is ranked against rankers
// trained only on strictly-earlier events. This scenario is constructed so the
// GBUS ranker beats the baseline: "noise" accumulates passive resource_saved
// signals (which the plain baseline rewards but GBUS confidence-discounts to 0),
// while the user's actual filings go to "good".
func TestEvaluate_GBUSBeatsBaseline(t *testing.T) {
	events := []gbus.GBUSSignalPayload{
		ev(1, gbus.SignalResourceSaved, "noise"),       // passive history favouring noise
		ev(2, gbus.SignalResourceSaved, "noise"),       //
		ev(3, gbus.SignalResourceSaved, "noise"),       //
		ev(4, gbus.SignalManualClassification, "good"), // baseline ranks noise top → miss; gbus: noise=0 → hit
		ev(5, gbus.SignalManualClassification, "good"), // both rankers now see good → hit
		ev(6, gbus.SignalManualClassification, "good"), // both hit
	}
	m := evaluate(events)
	if m.testCount != 3 {
		t.Fatalf("testCount = %d, want 3 (the manual 'good' instances)", m.testCount)
	}
	// GBUS hits all 3 (passive noise is discounted from instance 4 onward);
	// baseline misses instance 4 (passive noise outranks good) → 2/3.
	if m.gbusTop1 != 1.0 {
		t.Errorf("gbusTop1 = %.4f, want 1.0", m.gbusTop1)
	}
	if got := m.baseTop1; got < 0.66 || got > 0.67 {
		t.Errorf("baseTop1 = %.4f, want ~0.6667", got)
	}
	if m.gbusTop1 <= m.baseTop1 {
		t.Errorf("GBUS should beat baseline here (gbus=%.3f base=%.3f)", m.gbusTop1, m.baseTop1)
	}
}

func TestShouldPromote_Gate(t *testing.T) {
	// Beats baseline by >5% with enough instances → promote.
	if !shouldPromote(metrics{gbusTop1: 0.80, baseTop1: 0.70, testCount: 20}) {
		t.Error("clear lift with enough instances should promote")
	}
	// Lift below threshold → no promote.
	if shouldPromote(metrics{gbusTop1: 0.72, baseTop1: 0.70, testCount: 20}) {
		t.Error("sub-5% lift should not promote")
	}
	// Enough lift but too few instances → no promote.
	if shouldPromote(metrics{gbusTop1: 0.90, baseTop1: 0.50, testCount: 3}) {
		t.Error("too few test instances should not promote")
	}
	// GBUS worse than baseline → no promote.
	if shouldPromote(metrics{gbusTop1: 0.60, baseTop1: 0.70, testCount: 50}) {
		t.Error("GBUS below baseline should not promote")
	}
}