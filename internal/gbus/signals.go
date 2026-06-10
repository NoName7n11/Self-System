package gbus

import "time"

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

// GBUSSignalPayload is the event payload for every GBUS interaction signal.
// Stored as events with aggregate_type = "gbus_signal".
type GBUSSignalPayload struct {
	SignalType string    `json:"signal_type"`
	ResourceID string    `json:"resource_id,omitempty"`
	CategoryID string    `json:"category_id,omitempty"`
	Weight     float64   `json:"weight"`
	Context    string    `json:"context,omitempty"`
	OccurredAt time.Time `json:"occurred_at"`
}
