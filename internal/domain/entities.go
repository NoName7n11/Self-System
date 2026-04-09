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

type Resource struct {
	ID           string
	URL          string
	Host         string
	Title        string
	Summary      string
	CategoryID   string
	CategoryName string
	UserOverride bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
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

type ChatCommand struct {
	Message string
}
