package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"selfsystems/internal/domain"
)

type TodoRepository struct {
	db *sql.DB
}

func NewTodoRepository(db *sql.DB) *TodoRepository {
	return &TodoRepository{db: db}
}

func (r *TodoRepository) Create(ctx context.Context, todo *domain.Todo) error {
	timestamp := nowRFC3339()
	todo.CreatedAt = parseRFC3339(timestamp)
	todo.UpdatedAt = todo.CreatedAt

	var dueAt *string
	if todo.DueAt != nil {
		formatted := todo.DueAt.UTC().Format(timeLayout)
		dueAt = &formatted
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO todos (id, title, details, status, due_at, resource_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, todo.ID, todo.Title, todo.Details, string(todo.Status), dueAt, todo.ResourceID, timestamp, timestamp)
	if err != nil {
		return fmt.Errorf("create todo: %w", err)
	}

	return nil
}

func (r *TodoRepository) List(ctx context.Context, limit, offset int) ([]domain.Todo, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, details, status, due_at, resource_id, created_at, updated_at
		FROM todos
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list todos: %w", err)
	}
	defer rows.Close()

	items := make([]domain.Todo, 0)
	for rows.Next() {
		var item domain.Todo
		var dueAt sql.NullString
		var resourceID sql.NullString
		var createdAt string
		var updatedAt string
		var status string
		if err := rows.Scan(
			&item.ID,
			&item.Title,
			&item.Details,
			&status,
			&dueAt,
			&resourceID,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan todo: %w", err)
		}

		item.Status = domain.TodoStatus(status)
		if dueAt.Valid {
			t := parseRFC3339(dueAt.String)
			item.DueAt = &t
		}
		if resourceID.Valid {
			id := resourceID.String
			item.ResourceID = &id
		}
		item.CreatedAt = parseRFC3339(createdAt)
		item.UpdatedAt = parseRFC3339(updatedAt)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate todos: %w", err)
	}

	return items, nil
}
