package domain

import "time"

type CategorySource string

const (
	CategorySourceAuto   CategorySource = "auto"
	CategorySourceManual CategorySource = "manual"
)

type Category struct {
	ID            string
	Name          string
	Description   string
	Source        CategorySource
	AcceptCount   int
	OverrideCount int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ResourceExtractedData holds content extracted by the skim and deep processing
// tiers. It is stored as a JSON blob in the resources table and populated
// asynchronously after the resource is created.
type ResourceExtractedData struct {
	// Skim tier (URL extractor)
	ExtractedTitle       string `json:"extracted_title,omitempty"`
	ExtractedDescription string `json:"extracted_description,omitempty"`
	MainText             string `json:"main_text,omitempty"`
	PageType             string `json:"page_type,omitempty"`

	// Classification (Change 7 WS1)
	ClassificationConfidence float64 `json:"classification_confidence,omitempty"`
	ClassificationSource     string  `json:"classification_source,omitempty"` // "ai" | "heuristic"
	NeedsReview              bool    `json:"needs_review,omitempty"`          // true when confidence < threshold

	// Deep tier (AI enrichment — Change 7)
	KeyPoints []string `json:"key_points,omitempty"`
	Entities  []string `json:"entities,omitempty"`
	EventDate string   `json:"event_date,omitempty"`
	Deadline  string   `json:"deadline,omitempty"`
	Location  string   `json:"location,omitempty"`

	// Image tier (Change 6 WS3)
	ImageType       string `json:"image_type,omitempty"`
	ImageFormat     string `json:"image_format,omitempty"`
	ImageWidth      int    `json:"image_width,omitempty"`
	ImageHeight     int    `json:"image_height,omitempty"`
	ThumbnailBase64 string `json:"thumbnail_base64,omitempty"` // PNG thumbnail, base64-encoded
	OCRText         string `json:"ocr_text,omitempty"`         // populated in Change 7

	// PDF tier (Change 6 WS2)
	PDFPageCount int    `json:"pdf_page_count,omitempty"`
	PDFText      string `json:"pdf_text,omitempty"`
}

// ArchiveReason identifies why a resource was archived.
type ArchiveReason string

const (
	ArchiveReasonManual   ArchiveReason = "manual"
	ArchiveReasonDeadLink ArchiveReason = "dead_link"
	ArchiveReasonExpired  ArchiveReason = "expired"
)

type Resource struct {
	ID            string
	URL           string
	Host          string
	Title         string
	Summary       string
	CategoryID    string
	CategoryName  string
	UserOverride  bool
	ExtractedData ResourceExtractedData
	// SaveCount tracks how many times this URL has been submitted. Starts at 1.
	SaveCount int `json:"save_count"`
	// Archived is true when the resource has been soft-archived.
	Archived      bool          `json:"archived"`
	ArchiveReason ArchiveReason `json:"archive_reason"`
	ArchivedAt    *time.Time    `json:"archived_at,omitempty"`
	// SimilarTo holds resource IDs that are content-similar (cosine > threshold).
	SimilarTo []string `json:"similar_to,omitempty"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// SimilarResource links two resources that share high embedding similarity.
type SimilarResource struct {
	ResourceID      string
	SimilarID       string
	SimilarityScore float64
	CreatedAt       time.Time
}

type TodoStatus string

const (
	TodoStatusOpen       TodoStatus = "open"
	TodoStatusInProgress TodoStatus = "in_progress"
	TodoStatusDone       TodoStatus = "done"
)

type Todo struct {
	ID         string
	Title      string
	Details    string
	Status     TodoStatus
	DueAt      *time.Time
	ResourceID *string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type ReminderStatus string

const (
	ReminderStatusScheduled ReminderStatus = "scheduled"
	ReminderStatusSent      ReminderStatus = "sent"
	ReminderStatusDismissed ReminderStatus = "dismissed"
)

type Reminder struct {
	ID         string
	Title      string
	Message    string
	RemindAt   time.Time
	Status     ReminderStatus
	ResourceID *string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// ResourceEmbedding is a stored vector representation of a resource's content,
// tagged with the model version that produced it (Change 7 WS2/WS3).
type ResourceEmbedding struct {
	ResourceID   string
	Vector       []float32
	ModelVersion string
	Dim          int
	CreatedAt    time.Time
}

type ChatCommand struct {
	Message string
}

// GBUSCategoryFeature holds aggregated interaction weights for a category.
type GBUSCategoryFeature struct {
	CategoryID   string
	SignalType   string
	TotalWeight  float64
	SignalCount  int
	LastSignalAt time.Time
}

// GBUSResourceFeature holds aggregated interaction weights for a single resource.
type GBUSResourceFeature struct {
	ResourceID   string
	SignalType   string
	TotalWeight  float64
	SignalCount  int
	LastSignalAt time.Time
}
