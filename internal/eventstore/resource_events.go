package eventstore

import "time"

const (
	AggregateTypeResource = "resource"

	EventTypeResourceCreated          = "ResourceCreated"
	EventTypeResourceImported         = "ResourceImported" // backfill of pre-existing row
	EventTypeResourceUpdated          = "ResourceUpdated"
	EventTypeResourceDeleted          = "ResourceDeleted"
	EventTypeResourceSummaryUpdated   = "ResourceSummaryUpdated"
	EventTypeResourceCategoryAssigned = "ResourceCategoryAssigned"
	EventTypeResourceArchived         = "ResourceArchived"
	EventTypeResourceUnarchived       = "ResourceUnarchived"
	EventTypeResourceSkimCompleted    = "ResourceSkimCompleted"
	EventTypeResourcePDFExtracted     = "ResourcePDFExtracted"
	EventTypeResourceImageProcessed   = "ResourceImageProcessed"
	EventTypeResourceEventDetected    = "ResourceEventDetected"
	EventTypeResourceClassified       = "ResourceClassified"
	EventTypeResourceEmbedded           = "ResourceEmbedded"
	EventTypeResourceEnriched           = "ResourceEnriched"
	EventTypeResourceCounterIncremented = "ResourceCounterIncremented"
	EventTypeResourceSimilarityDetected = "ResourceSimilarityDetected"
	EventTypeResourceRestored           = "ResourceRestored"
)

// ResourceCreatedPayload is the v1 payload for ResourceCreated events.
type ResourceCreatedPayload struct {
	URL          string    `json:"url"`
	Host         string    `json:"host"`
	Title        string    `json:"title"`
	Summary      string    `json:"summary"`
	CategoryID   string    `json:"category_id"`
	CategoryName string    `json:"category_name"`
	UserOverride bool      `json:"user_override"`
	// ExtractedDataJSON carries the resource's extracted_data blob (classification
	// confidence/source/needs_review set at create time). Optional and additive —
	// empty for events written before Change 7. Projectors write it verbatim into
	// the extracted_data column, defaulting to "{}" when empty.
	ExtractedDataJSON string    `json:"extracted_data_json,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// ResourceUpdatedPayload is the v1 payload for ResourceUpdated events.
type ResourceUpdatedPayload struct {
	URL          string    `json:"url"`
	Host         string    `json:"host"`
	Title        string    `json:"title"`
	Summary      string    `json:"summary"`
	CategoryID   string    `json:"category_id"`
	CategoryName string    `json:"category_name"`
	UserOverride bool      `json:"user_override"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ResourceDeletedPayload is the v1 payload for ResourceDeleted events.
type ResourceDeletedPayload struct {
	ID string `json:"id"`
}

// ResourceCategoryAssignedPayload is the v1 payload for ResourceCategoryAssigned events.
type ResourceCategoryAssignedPayload struct {
	CategoryID   string    `json:"category_id"`
	CategoryName string    `json:"category_name"`
	UserOverride bool      `json:"user_override"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ResourcePDFExtractedPayload is the v1 payload for ResourcePDFExtracted events.
// Emitted by the deep processing worker after PDF text extraction completes.
type ResourcePDFExtractedPayload struct {
	PageCount   int    `json:"page_count"`
	SizeClass   string `json:"size_class"`
	TextLength  int    `json:"text_length"`
	CompletedAt string `json:"completed_at"`
}

// ResourceImageProcessedPayload is the v1 payload for ResourceImageProcessed events.
// Emitted by the deep processing worker after image classification and thumbnail generation.
type ResourceImageProcessedPayload struct {
	Format      string `json:"format"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	ImageType   string `json:"image_type"`
	HasThumbnail bool  `json:"has_thumbnail"`
	CompletedAt string `json:"completed_at"`
}

// ResourceEnrichedPayload is the v1 payload for ResourceEnriched events.
// Emitted by the deep processing worker when AI enrichment completes.
type ResourceEnrichedPayload struct {
	Summary   string   `json:"summary"`
	KeyPoints []string `json:"key_points"`
	Entities  []string `json:"entities"`
	Provider  string   `json:"provider"`
}

// ResourceClassifiedPayload is the v1 payload for ResourceClassified events.
// Emitted when a resource is classified into a category, carrying the confidence
// score, the source (ai/heuristic), and whether it was flagged for review.
type ResourceClassifiedPayload struct {
	CategoryID   string  `json:"category_id"`
	CategoryName string  `json:"category_name"`
	Confidence   float64 `json:"confidence"`
	Source       string  `json:"source"` // "ai" | "heuristic"
	NeedsReview  bool    `json:"needs_review"`
	ClassifiedAt string  `json:"classified_at"` // RFC3339
}

// ResourceEventDetectedPayload is the v1 payload for ResourceEventDetected events.
// Emitted when the event detector finds actionable signals (deadline, apply-by, etc.)
// in a resource's extracted text. ReminderID is set if a reminder was auto-created.
type ResourceEventDetectedPayload struct {
	Keyword    string `json:"keyword"`
	DateText   string `json:"date_text,omitempty"`
	EventDate  string `json:"event_date,omitempty"` // RFC3339, empty if no date found
	ReminderID string `json:"reminder_id,omitempty"`
}

// ResourceSkimCompletedPayload is the v1 payload for ResourceSkimCompleted events.
// Emitted by the async URL extractor after the skim pass populates extracted_data.
type ResourceSkimCompletedPayload struct {
	ExtractedTitle       string    `json:"extracted_title,omitempty"`
	ExtractedDescription string    `json:"extracted_description,omitempty"`
	PageType             string    `json:"page_type,omitempty"`
	TitleUpdated         bool      `json:"title_updated"` // true if resource.Title was replaced by extracted title
	CompletedAt          time.Time `json:"completed_at"`
}

// ResourceCounterIncrementedPayload is emitted when a duplicate URL save increments
// the save_count on an existing resource instead of creating a new record.
type ResourceCounterIncrementedPayload struct {
	NewCount int `json:"new_count"`
}

// ResourceArchivedPayload is emitted when a resource is soft-archived.
type ResourceArchivedPayload struct {
	Reason     string    `json:"reason"` // "manual" | "dead_link" | "expired"
	ArchivedAt time.Time `json:"archived_at"`
}

// ResourceRestoredPayload is emitted when an archived resource is restored.
type ResourceRestoredPayload struct {
	RestoredAt time.Time `json:"restored_at"`
}

// ResourceSimilarityDetectedPayload is emitted when a resource is found to be
// content-similar to an existing resource via embedding cosine distance.
type ResourceSimilarityDetectedPayload struct {
	SimilarResourceID string  `json:"similar_resource_id"`
	SimilarityScore   float64 `json:"similarity_score"`
}
