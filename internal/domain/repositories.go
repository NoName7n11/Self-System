package domain

import "context"

type CategoryRepository interface {
	List(ctx context.Context) ([]Category, error)
	GetByID(ctx context.Context, id string) (*Category, error)
	GetByName(ctx context.Context, name string) (*Category, error)
	Create(ctx context.Context, c *Category) error
	IncrementAccept(ctx context.Context, id string) error
	IncrementOverride(ctx context.Context, id string) error
}

type ResourceRepository interface {
	Create(ctx context.Context, r *Resource) error
	List(ctx context.Context, limit, offset int) ([]Resource, error)
	Search(ctx context.Context, query string, limit int) ([]Resource, error)
	UpdateCategory(ctx context.Context, resourceID, categoryID string, userOverride bool) error
}

type TodoRepository interface {
	Create(ctx context.Context, t *Todo) error
	List(ctx context.Context, limit, offset int) ([]Todo, error)
}

type ReminderRepository interface {
	Create(ctx context.Context, r *Reminder) error
	List(ctx context.Context, limit, offset int) ([]Reminder, error)
}
