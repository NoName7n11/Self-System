package eventstore

import "time"

const (
	AggregateTypeTodo = "todo"

	EventTypeTodoCreated = "TodoCreated"
	EventTypeTodoUpdated = "TodoUpdated"
	EventTypeTodoDeleted = "TodoDeleted"
)

// TodoCreatedPayload is the v1 payload for TodoCreated.
type TodoCreatedPayload struct {
	Title      string     `json:"title"`
	Details    string     `json:"details"`
	Status     string     `json:"status"`
	DueAt      *time.Time `json:"due_at,omitempty"`
	ResourceID *string    `json:"resource_id,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// TodoUpdatedPayload is the v1 payload for TodoUpdated.
type TodoUpdatedPayload struct {
	Title      string     `json:"title"`
	Details    string     `json:"details"`
	Status     string     `json:"status"`
	DueAt      *time.Time `json:"due_at,omitempty"`
	ResourceID *string    `json:"resource_id,omitempty"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// TodoDeletedPayload is the v1 payload for TodoDeleted.
type TodoDeletedPayload struct {
	ID string `json:"id"`
}
