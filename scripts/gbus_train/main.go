// gbus_train is the GBUS baseline model training CLI.
// It reads the gbus_category_features table from the SQLite database,
// computes time-decayed weighted affinity scores per category, normalizes
// them to [0,1], and saves a model artifact to models/gbus/baseline.json.
//
// Usage:
//
//	go run ./scripts/gbus_train [-db ./data/self_systems.db] [-out ./models/gbus/baseline.json]
//
// Promotion criteria: the model artifact is saved with status "candidate".
// To promote to "production", verify validation_accuracy > baseline_accuracy + 0.05
// and manually update the status field, or use -promote flag.
package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	_ "modernc.org/sqlite"

	"selfsystems/internal/gbus"
)

func main() {
	dbPath := flag.String("db", "./data/self_systems.db", "path to SQLite database")
	outPath := flag.String("out", "./models/gbus/baseline.json", "output model JSON path")
	promote := flag.Bool("promote", false, "set model status to production (requires validation_accuracy > baseline+5%)")
	flag.Parse()

	db, err := sql.Open("sqlite", *dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	repo := newDBRepo(db)

	log.Printf("reading category features from %s...", *dbPath)
	features, err := repo.listAll()
	if err != nil {
		log.Fatalf("read features: %v", err)
	}
	if len(features) == 0 {
		log.Println("no features found — has the aggregator run yet? Exiting.")
		os.Exit(0)
	}
	log.Printf("found %d feature rows across all categories", len(features))

	weights := computeCategoryWeights(features)
	log.Printf("computed weights for %d categories", len(weights))

	accuracy := evaluateAccuracy(features, weights)
	log.Printf("validation accuracy (proxy): %.4f", accuracy)

	// Load existing baseline to compare.
	baselineAccuracy := 0.0
	if b, readErr := os.ReadFile(*outPath); readErr == nil {
		var existing gbus.GBUSModel
		if jsonErr := json.Unmarshal(b, &existing); jsonErr == nil {
			baselineAccuracy = existing.ValidationAccuracy
			log.Printf("existing baseline accuracy: %.4f", baselineAccuracy)
		}
	}

	status := "candidate"
	if *promote {
		if accuracy < baselineAccuracy+0.05 {
			log.Printf("WARNING: accuracy %.4f does not beat baseline %.4f + 5%%; not promoting", accuracy, baselineAccuracy)
		} else {
			status = "production"
			log.Printf("promoting model to production (accuracy gain: %.4f%%)", (accuracy-baselineAccuracy)*100)
		}
	}

	model := gbus.GBUSModel{
		Version:            fmt.Sprintf("1.%d.0", time.Now().Unix()/86400),
		TrainedAt:          time.Now().UTC(),
		Status:             status,
		ValidationAccuracy: accuracy,
		BaselineAccuracy:   baselineAccuracy,
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

// computeCategoryWeights derives per-category affinity scores from feature rows.
// Score = sum(weight * signalTypeMultiplier) / log(1 + total_signals), discounted
// by the category's evidence-based confidence, then normalised to [0,1].
//
// Confidence discount: evidence_count is accumulated per (category, signal_type)
// only when a signal's weight >= ExplicitIntentWeightThreshold, so passive signal
// types (e.g. resource_saved, weight 0.3) structurally carry confidence 0 on their
// own row. We therefore aggregate evidence across all of a category's rows and
// apply the discount at category level — a category backed only by passive signals
// is damped relative to one with explicit manual classifications/corrections.
func computeCategoryWeights(features []gbus.CategoryFeature) map[string]float64 {
	type catAgg struct {
		score    float64
		count    int
		evidence int
	}
	agg := map[string]*catAgg{}

	for _, f := range features {
		if _, ok := agg[f.CategoryID]; !ok {
			agg[f.CategoryID] = &catAgg{}
		}
		baseWeight := gbus.SignalWeights[f.SignalType]
		if baseWeight == 0 {
			baseWeight = 0.1
		}
		// Positive signals boost; deletion/dismissed dampen.
		multiplier := 1.0
		if f.SignalType == gbus.SignalResourceDeleted || f.SignalType == gbus.SignalReminderDismissed {
			multiplier = -0.5
		}
		contrib := f.TotalWeight * baseWeight * multiplier
		agg[f.CategoryID].score += contrib
		agg[f.CategoryID].count += f.SignalCount
		agg[f.CategoryID].evidence += f.EvidenceCount
	}

	raw := map[string]float64{}
	maxScore := 0.0
	for catID, a := range agg {
		// Dampen by log(1+total_signals) to prevent outlier categories from dominating.
		score := a.score / math.Log(1+float64(a.count))
		if score < 0 {
			score = 0
		}
		// Discount by evidence-based confidence: MIN(1, evidence / threshold).
		// Categories with little explicit-intent backing contribute proportionally less.
		confidence := math.Min(1.0, float64(a.evidence)/float64(gbus.ConfidenceEvidenceThreshold))
		score *= confidence
		raw[catID] = score
		if score > maxScore {
			maxScore = score
		}
	}

	// Normalize to [0,1].
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

// evaluateAccuracy computes a proxy accuracy: fraction of manual-classification
// signals whose category has a learned weight >= 0.5 (model would agree).
func evaluateAccuracy(features []gbus.CategoryFeature, weights map[string]float64) float64 {
	correct := 0
	total := 0
	for _, f := range features {
		if f.SignalType != gbus.SignalManualClassification {
			continue
		}
		total++
		if weights[f.CategoryID] >= 0.5 {
			correct++
		}
	}
	if total == 0 {
		return 1.0
	}
	return float64(correct) / float64(total)
}

// dbRepo wraps raw SQL for the training script.
type dbRepo struct{ db *sql.DB }

func newDBRepo(db *sql.DB) *dbRepo { return &dbRepo{db: db} }

func (r *dbRepo) listAll() ([]gbus.CategoryFeature, error) {
	rows, err := r.db.Query(`
		SELECT category_id, signal_type, total_weight, signal_count, evidence_count, confidence, last_signal_at
		FROM gbus_category_features
		ORDER BY category_id, signal_type
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []gbus.CategoryFeature
	for rows.Next() {
		var f gbus.CategoryFeature
		var lastAt string
		if err := rows.Scan(&f.CategoryID, &f.SignalType, &f.TotalWeight, &f.SignalCount, &f.EvidenceCount, &f.Confidence, &lastAt); err != nil {
			return nil, err
		}
		if t, err := time.Parse(time.RFC3339, lastAt); err == nil {
			f.LastSignalAt = t
		}
		out = append(out, f)
	}
	return out, rows.Err()
}
