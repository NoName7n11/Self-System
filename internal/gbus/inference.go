package gbus

import (
	"encoding/json"
	"log/slog"
	"math"
	"os"
	"sync"
	"time"
)

// GBUSModel is the serialized model artifact produced by the training pipeline.
type GBUSModel struct {
	Version            string             `json:"version"`
	TrainedAt          time.Time          `json:"trained_at"`
	Status             string             `json:"status"` // "candidate" | "production" | "retired"
	ValidationAccuracy float64            `json:"validation_accuracy"`
	BaselineAccuracy   float64            `json:"baseline_accuracy"`
	CategoryWeights    map[string]float64 `json:"category_weights"` // categoryID → affinity [0,1]
}

// Inference loads and applies a GBUS model artifact to provide category affinity
// scores and classification biasing.
type Inference struct {
	modelPath string
	model     *GBUSModel
	mu        sync.RWMutex
	enabled   bool
}

// NewInference creates an Inference engine. When modelPath is empty or
// enabled=false, all methods are safe no-ops.
func NewInference(modelPath string, enabled bool) *Inference {
	inf := &Inference{modelPath: modelPath, enabled: enabled}
	if enabled && modelPath != "" {
		inf.load()
	}
	return inf
}

func (inf *Inference) load() {
	b, err := os.ReadFile(inf.modelPath)
	if err != nil {
		slog.Warn("gbus inference: load model", "path", inf.modelPath, "error", err)
		return
	}
	var m GBUSModel
	if err := json.Unmarshal(b, &m); err != nil {
		slog.Warn("gbus inference: parse model", "path", inf.modelPath, "error", err)
		return
	}
	if m.Status != "production" {
		slog.Warn("gbus inference: model not in production status", "status", m.Status, "version", m.Version)
		return
	}
	inf.mu.Lock()
	inf.model = &m
	inf.mu.Unlock()
	slog.Info("gbus inference: model loaded", "version", m.Version, "categories", len(m.CategoryWeights))
}

// CategoryScore returns the GBUS affinity score [0,1] for a category.
// Returns 0 when the model is not loaded or the category has no signal history.
func (inf *Inference) CategoryScore(categoryID string) float64 {
	if !inf.enabled {
		return 0
	}
	inf.mu.RLock()
	defer inf.mu.RUnlock()
	if inf.model == nil {
		return 0
	}
	return inf.model.CategoryWeights[categoryID]
}

// BiasClassification applies a gentle GBUS interest boost to a classification
// confidence score. When the user has shown repeated interest in a category,
// the classifier confidence is nudged upward (max +10%).
func (inf *Inference) BiasClassification(categoryID string, confidence float64) float64 {
	score := inf.CategoryScore(categoryID)
	if score <= 0 {
		return confidence
	}
	boost := score * 0.10
	biased := confidence + boost
	if biased > 1.0 {
		return 1.0
	}
	return biased
}

// RerankByInterest reorders results by GBUS category affinity. Results whose
// category has high user interest score are promoted; ties preserve original order.
// Weight controls how strongly GBUS reranks vs original rank.
func (inf *Inference) RerankByInterest(ids []string, categoryIDs []string, weight float64) []int {
	if !inf.enabled || len(ids) == 0 {
		// identity permutation
		perm := make([]int, len(ids))
		for i := range perm {
			perm[i] = i
		}
		return perm
	}
	inf.mu.RLock()
	defer inf.mu.RUnlock()

	type scored struct {
		idx   int
		score float64
	}
	items := make([]scored, len(ids))
	for i, catID := range categoryIDs {
		gScore := 0.0
		if inf.model != nil {
			gScore = inf.model.CategoryWeights[catID]
		}
		// Blend original rank (descending from 1.0) with GBUS score.
		origRank := 1.0 - float64(i)/math.Max(1, float64(len(ids)))
		items[i] = scored{idx: i, score: origRank + gScore*weight}
	}
	// Stable sort by descending score.
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && items[j].score > items[j-1].score; j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
	perm := make([]int, len(items))
	for i, it := range items {
		perm[i] = it.idx
	}
	return perm
}

// ModelVersion returns the loaded model's version string, or "none" when not loaded.
func (inf *Inference) ModelVersion() string {
	inf.mu.RLock()
	defer inf.mu.RUnlock()
	if inf.model == nil {
		return "none"
	}
	return inf.model.Version
}

// ModelStatus returns the loaded model's status, or "not_loaded".
func (inf *Inference) ModelStatus() string {
	inf.mu.RLock()
	defer inf.mu.RUnlock()
	if inf.model == nil {
		return "not_loaded"
	}
	return inf.model.Status
}

// Reload re-reads the model file from disk. Called by the monitor when a new
// model artifact is promoted to production.
func (inf *Inference) Reload() {
	if inf.enabled && inf.modelPath != "" {
		inf.load()
	}
}
