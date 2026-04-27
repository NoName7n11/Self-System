package domain

import "context"

type CategoryRepository interface {
	List(ctx context.Context) ([]Category, error)
	GetByID(ctx context.Context, id string) (*Category, error)
	GetByName(ctx context.Context, name string) (*Category, error)
	Create(ctx context.Context, c *Category) error
	Update(ctx context.Context, c *Category) error
	Delete(ctx context.Context, id string) error
	IncrementAccept(ctx context.Context, id string) error
	IncrementOverride(ctx context.Context, id string) error
}

type ResourceRepository interface {
	GetByID(ctx context.Context, id string) (*Resource, error)
	Create(ctx context.Context, r *Resource) error
	Update(ctx context.Context, r *Resource) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]Resource, error)
	Search(ctx context.Context, query string, limit int) ([]Resource, error)
	UpdateCategory(ctx context.Context, resourceID, categoryID string, userOverride bool) error
}

type TodoRepository interface {
	GetByID(ctx context.Context, id string) (*Todo, error)
	Create(ctx context.Context, t *Todo) error
	Update(ctx context.Context, t *Todo) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]Todo, error)
}

type ReminderRepository interface {
	GetByID(ctx context.Context, id string) (*Reminder, error)
	Create(ctx context.Context, r *Reminder) error
	Update(ctx context.Context, r *Reminder) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]Reminder, error)
}
