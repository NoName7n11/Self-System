package eventstore

import "time"

const (
	AggregateTypeCategory = "category"

	EventTypeCategoryCreated = "CategoryCreated"
	EventTypeCategoryUpdated = "CategoryUpdated"
	EventTypeCategoryDeleted = "CategoryDeleted"
)

// CategoryCreatedPayload is the v1 payload for CategoryCreated.
type CategoryCreatedPayload struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Source      string    `json:"source"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CategoryUpdatedPayload is the v1 payload for CategoryUpdated.
type CategoryUpdatedPayload struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CategoryDeletedPayload is the v1 payload for CategoryDeleted.
type CategoryDeletedPayload struct {
	ID string `json:"id"`
}
