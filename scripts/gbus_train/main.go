// gbus_train is the GBUS baseline model training CLI.
//
// It reads raw GBUS signal events from the event log (aggregate_type =
// "gbus_signal"), computes a category-affinity prior, and saves a model
// artifact to models/gbus/baseline.json.
//
// GBUS produces a GLOBAL category-affinity prior, not a per-resource
// classifier. The honest offline metric is therefore a ranking metric: on
// held-out future "user chose category C" events (manual_classification /
// category_correction), does the prior rank C in its top-1 / top-3? We compare
// two rankers built from the same events:
//
//   - baseline: the Outline weighted-scoring scheme — a plain sum of signal
//     weights per category (no dampening, no confidence, no decay).
//   - gbus:     the enhanced ranker — log-dampened, delete/dismiss-penalised,
//     and discounted by evidence-based confidence (Change 16 WS6).
//
// Evaluation is prequential (expanding-window): each labeled instance, in
// chronological order, is scored against a ranker trained ONLY on strictly
// earlier events. This is the small-data-robust form of a temporal holdout —
// no future leakage, uses every instance, and degrades gracefully on tiny
// datasets. Both rankers suffer cold-start equally, so the lift is fair.
//
// Usage:
//
//	go run ./scripts/gbus_train [-db ./data/self_systems.db] [-out ./models/gbus/baseline.json] [-promote]
//
// The model artifact is saved with status "candidate". With -promote, the
// status is set to "production" only when the GBUS top-1 beats the baseline
// top-1 by >= 5% AND there are enough test instances to trust the number.
package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"time"

	_ "modernc.org/sqlite"

	"selfsystems/internal/gbus"
)

// promotionLiftThreshold is the minimum top-1 accuracy gain over the weighted-
// scoring baseline required to promote a model to production.
const promotionLiftThreshold = 0.05

// minTestForPromote guards against promoting on a handful of noisy instances.
const minTestForPromote = 10

func main() {
	dbPath := flag.String("db", "./data/self_systems.db", "path to SQLite database")
	outPath := flag.String("out", "./models/gbus/baseline.json", "output model JSON path")
	promote := flag.Bool("promote", false, "set model status to production (requires GBUS top-1 > baseline top-1 + 5%)")
	flag.Parse()

	db, err := sql.Open("sqlite", *dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	repo := newDBRepo(db)

	log.Printf("reading gbus signal events from %s...", *dbPath)
	events, err := repo.listSignalEvents()
	if err != nil {
		log.Fatalf("read signal events: %v", err)
	}
	if len(events) == 0 {
		log.Println("no gbus signal events found — is gbus.enabled and has the app been used? Exiting without writing a model.")
		os.Exit(0)
	}
	log.Printf("found %d signal events with a category", len(events))

	// Held-out evaluation (prequential). Reports real, comparable numbers.
	metrics := evaluate(events)
	if metrics.testCount == 0 {
		log.Println("no labeled instances (manual_classification / category_correction) to evaluate — writing model as candidate without accuracy.")
	} else {
		log.Printf("evaluation over %d held-out instances:", metrics.testCount)
		log.Printf("  GBUS     top-1=%.4f top-3=%.4f", metrics.gbusTop1, metrics.gbusTop3)
		log.Printf("  baseline top-1=%.4f top-3=%.4f", metrics.baseTop1, metrics.baseTop3)
		log.Printf("  top-1 lift over weighted scoring: %+.4f", metrics.gbusTop1-metrics.baseTop1)
	}

	// Shipped weights are computed from ALL events (production uses every signal).
	weights := gbusWeights(events)
	log.Printf("computed shipped weights for %d categories", len(weights))

	status := "candidate"
	if *promote {
		if shouldPromote(metrics) {
			status = "production"
			log.Printf("promoting to production (top-1 lift: %+.4f)", metrics.gbusTop1-metrics.baseTop1)
		} else if metrics.testCount < minTestForPromote {
			log.Printf("WARNING: only %d test instances (< %d); not promoting", metrics.testCount, minTestForPromote)
		} else {
			log.Printf("WARNING: GBUS top-1 %.4f does not beat baseline %.4f by %.0f%%; not promoting",
				metrics.gbusTop1, metrics.baseTop1, promotionLiftThreshold*100)
		}
	}

	model := gbus.GBUSModel{
		Version:            fmt.Sprintf("1.%d.0", time.Now().Unix()/86400),
		TrainedAt:          time.Now().UTC(),
		Status:             status,
		ValidationAccuracy: metrics.gbusTop1,
		BaselineAccuracy:   metrics.baseTop1,
		CategoryWeights:    weights,
	}

	b, err := json.MarshalIndent(model, "", "  ")
	if err != nil {
		log.Fatalf("marshal model: %v", err)
	}
	if err := os.WriteFile(*outPath, b, 0644); err != nil {
		log.Fatalf("write model: %v", err)
	}
	log.Printf("model saved to %s (status=%s, version=%s, categories=%d)",
		*outPath, model.Status, model.Version, len(weights))
}

// shouldPromote reports whether the evaluated model is good enough to promote
// to production: enough held-out instances to trust the number, and a GBUS top-1
// that beats the weighted-scoring baseline top-1 by at least the lift threshold.
func shouldPromote(m metrics) bool {
	if m.testCount < minTestForPromote {
		return false
	}
	return m.gbusTop1 >= m.baseTop1+promotionLiftThreshold
}

// baselineWeights is the Outline weighted-scoring scheme: a plain sum of signal
// base weights per category, normalised to [0,1]. This is the baseline the
// GBUS model must beat.
func baselineWeights(events []gbus.GBUSSignalPayload) map[string]float64 {
	raw := map[string]float64{}
	maxScore := 0.0
	for _, e := range events {
		if e.CategoryID == "" {
			continue
		}
		w := gbus.SignalWeights[e.SignalType]
		if w == 0 {
			w = 0.1
		}
		raw[e.CategoryID] += w
		if raw[e.CategoryID] > maxScore {
			maxScore = raw[e.CategoryID]
		}
	}
	return normalize(raw, maxScore)
}

// gbusWeights is the enhanced GBUS ranker: per-category interaction score
// dampened by log(1+count), penalising deletions/dismissals, then discounted by
// evidence-based confidence (MIN(1, evidence/threshold)), normalised to [0,1].
// evidence_count is recomputed in-memory from the events here (the aggregator's
// ExplicitIntentWeightThreshold rule) so there is no leakage from stored
// aggregate tables.
func gbusWeights(events []gbus.GBUSSignalPayload) map[string]float64 {
	type catAgg struct {
		score    float64
		count    int
		evidence int
	}
	agg := map[string]*catAgg{}

	for _, e := range events {
		if e.CategoryID == "" {
			continue
		}
		if _, ok := agg[e.CategoryID]; !ok {
			agg[e.CategoryID] = &catAgg{}
		}
		baseWeight := gbus.SignalWeights[e.SignalType]
		if baseWeight == 0 {
			baseWeight = 0.1
		}
		multiplier := 1.0
		if e.SignalType == gbus.SignalResourceDeleted || e.SignalType == gbus.SignalReminderDismissed {
			multiplier = -0.5
		}
		a := agg[e.CategoryID]
		a.score += e.Weight * baseWeight * multiplier
		a.count++
		if e.Weight >= gbus.ExplicitIntentWeightThreshold {
			a.evidence++
		}
	}

	raw := map[string]float64{}
	maxScore := 0.0
	for catID, a := range agg {
		score := a.score / math.Log(1+float64(a.count))
		if score < 0 {
			score = 0
		}
		// Discount by evidence-based confidence: categories with little
		// explicit-intent backing contribute proportionally less.
		confidence := math.Min(1.0, float64(a.evidence)/float64(gbus.ConfidenceEvidenceThreshold))
		score *= confidence
		raw[catID] = score
		if score > maxScore {
			maxScore = score
		}
	}
	return normalize(raw, maxScore)
}

func normalize(raw map[string]float64, maxScore float64) map[string]float64 {
	result := make(map[string]float64, len(raw))
	for catID, score := range raw {
		if maxScore > 0 {
			result[catID] = score / maxScore
		} else {
			result[catID] = 0
		}
	}
	return result
}

// metrics holds top-1/top-3 accuracy for both rankers over the held-out set.
type metrics struct {
	gbusTop1, gbusTop3 float64
	baseTop1, baseTop3 float64
	testCount          int
}

// isLabeled reports whether a signal is a ground-truth "user chose category C"
// instance usable as a test label.
func isLabeled(signalType string) bool {
	return signalType == gbus.SignalManualClassification || signalType == gbus.SignalCategoryCorrection
}

// evaluate runs prequential (expanding-window) evaluation: each labeled instance,
// in chronological order, is ranked against rankers trained only on strictly
// earlier events. Candidate categories are the global universe seen across all
// events. Returns top-1/top-3 accuracy for the GBUS and baseline rankers.
func evaluate(events []gbus.GBUSSignalPayload) metrics {
	sorted := make([]gbus.GBUSSignalPayload, len(events))
	copy(sorted, events)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].OccurredAt.Before(sorted[j].OccurredAt)
	})

	// Global candidate universe.
	candidates := map[string]struct{}{}
	for _, e := range sorted {
		if e.CategoryID != "" {
			candidates[e.CategoryID] = struct{}{}
		}
	}

	var m metrics
	var gTop1, gTop3, bTop1, bTop3 int
	for i, e := range sorted {
		if !isLabeled(e.SignalType) || e.CategoryID == "" {
			continue
		}
		// Train on all events strictly before this instance's time.
		var train []gbus.GBUSSignalPayload
		for j := 0; j < i; j++ {
			if sorted[j].OccurredAt.Before(e.OccurredAt) {
				train = append(train, sorted[j])
			}
		}
		m.testCount++

		gw := gbusWeights(train)
		bw := baselineWeights(train)
		if hitAtK(e.CategoryID, gw, candidates, 1) {
			gTop1++
		}
		if hitAtK(e.CategoryID, gw, candidates, 3) {
			gTop3++
		}
		if hitAtK(e.CategoryID, bw, candidates, 1) {
			bTop1++
		}
		if hitAtK(e.CategoryID, bw, candidates, 3) {
			bTop3++
		}
	}

	if m.testCount > 0 {
		n := float64(m.testCount)
		m.gbusTop1 = float64(gTop1) / n
		m.gbusTop3 = float64(gTop3) / n
		m.baseTop1 = float64(bTop1) / n
		m.baseTop3 = float64(bTop3) / n
	}
	return m
}

// hitAtK reports whether trueCat ranks within the top k by weight among the
// candidate categories. Ties are handled fairly: a hit requires that fewer than
// k categories score STRICTLY higher than trueCat (so trueCat counts as top-1
// when nothing strictly beats it).
func hitAtK(trueCat string, weights map[string]float64, candidates map[string]struct{}, k int) bool {
	target := weights[trueCat]
	strictlyBetter := 0
	for cat := range candidates {
		if cat == trueCat {
			continue
		}
		if weights[cat] > target {
			strictlyBetter++
		}
	}
	return strictlyBetter < k
}

// dbRepo wraps raw SQL for the training script.
type dbRepo struct{ db *sql.DB }

func newDBRepo(db *sql.DB) *dbRepo { return &dbRepo{db: db} }

// listSignalEvents reads all gbus_signal events from the event log, in
// chronological order, returning the parsed payloads that carry a category.
func (r *dbRepo) listSignalEvents() ([]gbus.GBUSSignalPayload, error) {
	rows, err := r.db.Query(`
		SELECT payload, occurred_at
		FROM events
		WHERE aggregate_type = 'gbus_signal'
		ORDER BY occurred_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []gbus.GBUSSignalPayload
	for rows.Next() {
		var payload, occurredAt string
		if err := rows.Scan(&payload, &occurredAt); err != nil {
			return nil, err
		}
		var p gbus.GBUSSignalPayload
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			log.Printf("skip unparseable signal payload: %v", err)
			continue
		}
		if p.CategoryID == "" {
			continue
		}
		// Prefer the payload's occurred_at; fall back to the column.
		if p.OccurredAt.IsZero() {
			if t, perr := time.Parse(time.RFC3339, occurredAt); perr == nil {
				p.OccurredAt = t
			}
		}
		out = append(out, p)
	}
	return out, rows.Err()
}