package eventstore

import "time"

const (
	AggregateTypeReminder = "reminder"

	EventTypeReminderCreated = "ReminderCreated"
	EventTypeReminderUpdated = "ReminderUpdated"
	EventTypeReminderDeleted = "ReminderDeleted"
)

// ReminderCreatedPayload is the v1 payload for ReminderCreated.
type ReminderCreatedPayload struct {
	Title      string    `json:"title"`
	Message    string    `json:"message"`
	RemindAt   time.Time `json:"remind_at"`
	Status     string    `json:"status"`
	ResourceID *string   `json:"resource_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ReminderUpdatedPayload is the v1 payload for ReminderUpdated.
type ReminderUpdatedPayload struct {
	Title      string    `json:"title"`
	Message    string    `json:"message"`
	RemindAt   time.Time `json:"remind_at"`
	Status     string    `json:"status"`
	ResourceID *string   `json:"resource_id,omitempty"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ReminderDeletedPayload is the v1 payload for ReminderDeleted.
type ReminderDeletedPayload struct {
	ID string `json:"id"`
}
