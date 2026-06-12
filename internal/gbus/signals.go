package gbus

import (
	"time"

	"selfsystems/internal/domain"
)

// Signal type constants — correspond to the GBUS weighted scheme from the Outline.
const (
	SignalManualClassification = "manual_classification"  // weight 1.0
	SignalCategoryCorrection   = "category_correction"    // weight 1.0
	SignalAutoClassification   = "auto_classification"    // weight 0.5
	SignalResourceSaved        = "resource_saved"         // weight 0.3
	SignalResourceDeleted      = "resource_deleted"       // weight 0.1
	SignalResourceRevisited    = "resource_revisited"     // weight 0.4
	SignalCounterIncremented   = "counter_incremented"    // weight 0.2
	SignalSearchQuery          = "search_query"           // weight 0.2
	SignalReminderDismissed    = "reminder_dismissed"     // weight 0.1
	SignalDeepProcessConfirmed = "deep_process_confirmed" // weight 0.3
)

// SignalWeights maps each signal type to its base interaction weight.
var SignalWeights = map[string]float64{
	SignalManualClassification: 1.0,
	SignalCategoryCorrection:   1.0,
	SignalAutoClassification:   0.5,
	SignalResourceSaved:        0.3,
	SignalResourceDeleted:      0.1,
	SignalResourceRevisited:    0.4,
	SignalCounterIncremented:   0.2,
	SignalSearchQuery:          0.2,
	SignalReminderDismissed:    0.1,
	SignalDeepProcessConfirmed: 0.3,
}

// DefaultUserID is the user_id stamped on signals and feature rows while the
// app is single-user (Phase 1-2). Keying signals/features by (user_id, ...)
// from the start avoids a data migration when multi-user/sync (Phase 2+)
// lands — see Plans/Progress_Changes/Change_16_Workstream.md.
const DefaultUserID = "local"

// ExplicitIntentWeightThreshold and ConfidenceEvidenceThreshold are aliases
// for the domain constants of the same name (defined there to avoid an
// import cycle: sqlite repo -> gbus -> eventstore -> sqlite repo).
const (
	ExplicitIntentWeightThreshold = domain.ExplicitIntentWeightThreshold
	ConfidenceEvidenceThreshold   = domain.ConfidenceEvidenceThreshold
)

// GBUSSignalPayload is the event payload for every GBUS interaction signal.
// Stored as events with aggregate_type = "gbus_signal".
type GBUSSignalPayload struct {
	SignalType string    `json:"signal_type"`
	UserID     string    `json:"user_id,omitempty"`
	SessionID  string    `json:"session_id,omitempty"`
	ResourceID string    `json:"resource_id,omitempty"`
	CategoryID string    `json:"category_id,omitempty"`
	Weight     float64   `json:"weight"`
	Context    string    `json:"context,omitempty"`
	OccurredAt time.Time `json:"occurred_at"`
}
