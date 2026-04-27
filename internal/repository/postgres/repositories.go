package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"selfsystems/internal/domain"
)

var ErrNotImplemented = errors.New("postgres repository method not implemented")

type CategoryRepository struct {
	db *sql.DB
}

var _ domain.CategoryRepository = (*CategoryRepository)(nil)

func NewCategoryRepository(db *sql.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) List(ctx context.Context) ([]domain.Category, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, description, source, accept_count, override_count, created_at, updated_at
		FROM categories
		ORDER BY lower(name) ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	items := make([]domain.Category, 0)
	for rows.Next() {
		item, err := scanCategory(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate categories: %w", err)
	}

	return items, nil
}

func (r *CategoryRepository) GetByID(ctx context.Context, id string) (*domain.Category, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, description, source, accept_count, override_count, created_at, updated_at
		FROM categories
		WHERE id = $1
	`, strings.TrimSpace(id))

	item, err := scanCategory(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get category by id: %w", err)
	}

	return item, nil
}

func (r *CategoryRepository) GetByName(ctx context.Context, name string) (*domain.Category, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, description, source, accept_count, override_count, created_at, updated_at
		FROM categories
		WHERE lower(name) = lower($1)
	`, strings.TrimSpace(name))

	item, err := scanCategory(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get category by name: %w", err)
	}

	return item, nil
}

func (r *CategoryRepository) Create(ctx context.Context, c *domain.Category) error {
	now := time.Now().UTC()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	c.UpdatedAt = now

	source := strings.TrimSpace(string(c.Source))
	if source == "" {
		source = string(domain.CategorySourceManual)
		c.Source = domain.CategorySource(source)
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO categories (id, name, description, source, accept_count, override_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, c.ID, strings.TrimSpace(c.Name), strings.TrimSpace(c.Description), source, c.AcceptCount, c.OverrideCount, c.CreatedAt, c.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create category: %w", err)
	}

	return nil
}

func (r *CategoryRepository) Update(ctx context.Context, c *domain.Category) error {
	now := time.Now().UTC()
	c.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, `
		UPDATE categories
		SET name = $1, description = $2, source = $3, updated_at = $4
		WHERE id = $5
	`, strings.TrimSpace(c.Name), strings.TrimSpace(c.Description), string(c.Source), c.UpdatedAt, strings.TrimSpace(c.ID))
	if err != nil {
		return fmt.Errorf("update category: %w", err)
	}

	return nil
}

func (r *CategoryRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM categories
		WHERE id = $1
	`, strings.TrimSpace(id))
	if err != nil {
		return fmt.Errorf("delete category: %w", err)
	}

	return nil
}

func (r *CategoryRepository) IncrementAccept(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE categories
		SET accept_count = accept_count + 1, updated_at = $1
		WHERE id = $2
	`, time.Now().UTC(), strings.TrimSpace(id))
	if err != nil {
		return fmt.Errorf("increment category accept: %w", err)
	}

	return nil
}

func (r *CategoryRepository) IncrementOverride(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE categories
		SET override_count = override_count + 1, updated_at = $1
		WHERE id = $2
	`, time.Now().UTC(), strings.TrimSpace(id))
	if err != nil {
		return fmt.Errorf("increment category override: %w", err)
	}

	return nil
}

type ResourceRepository struct {
	db *sql.DB
}

var _ domain.ResourceRepository = (*ResourceRepository)(nil)

func NewResourceRepository(db *sql.DB) *ResourceRepository {
	return &ResourceRepository{db: db}
}

func (r *ResourceRepository) GetByID(ctx context.Context, id string) (*domain.Resource, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT r.id, r.url, r.host, r.title, r.summary, r.category_id, c.name, r.user_override, r.created_at, r.updated_at
		FROM resources r
		JOIN categories c ON c.id = r.category_id
		WHERE r.id = $1
	`, strings.TrimSpace(id))

	item, err := scanResource(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get resource by id: %w", err)
	}

	return item, nil
}

func (r *ResourceRepository) Create(ctx context.Context, resource *domain.Resource) error {
	now := time.Now().UTC()
	if resource.CreatedAt.IsZero() {
		resource.CreatedAt = now
	}
	resource.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO resources (id, url, host, title, summary, category_id, user_override, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, resource.ID, strings.TrimSpace(resource.URL), strings.TrimSpace(resource.Host), strings.TrimSpace(resource.Title), strings.TrimSpace(resource.Summary), strings.TrimSpace(resource.CategoryID), resource.UserOverride, resource.CreatedAt, resource.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create resource: %w", err)
	}

	return nil
}

func (r *ResourceRepository) Update(ctx context.Context, resource *domain.Resource) error {
	resource.UpdatedAt = time.Now().UTC()

	_, err := r.db.ExecContext(ctx, `
		UPDATE resources
		SET url = $1, host = $2, title = $3, summary = $4, category_id = $5, user_override = $6, updated_at = $7
		WHERE id = $8
	`, strings.TrimSpace(resource.URL), strings.TrimSpace(resource.Host), strings.TrimSpace(resource.Title), strings.TrimSpace(resource.Summary), strings.TrimSpace(resource.CategoryID), resource.UserOverride, resource.UpdatedAt, strings.TrimSpace(resource.ID))
	if err != nil {
		return fmt.Errorf("update resource: %w", err)
	}

	return nil
}

func (r *ResourceRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM resources
		WHERE id = $1
	`, strings.TrimSpace(id))
	if err != nil {
		return fmt.Errorf("delete resource: %w", err)
	}

	return nil
}

func (r *ResourceRepository) List(ctx context.Context, limit, offset int) ([]domain.Resource, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT r.id, r.url, r.host, r.title, r.summary, r.category_id, c.name, r.user_override, r.created_at, r.updated_at
		FROM resources r
		JOIN categories c ON c.id = r.category_id
		ORDER BY r.created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list resources: %w", err)
	}
	defer rows.Close()

	items := make([]domain.Resource, 0)
	for rows.Next() {
		item, err := scanResource(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate resources: %w", err)
	}

	return items, nil
}

func (r *ResourceRepository) Search(ctx context.Context, query string, limit int) ([]domain.Resource, error) {
	if limit <= 0 {
		limit = 25
	}

	query = strings.TrimSpace(query)
	if query == "" {
		return []domain.Resource{}, nil
	}

	pattern := "%" + query + "%"
	rows, err := r.db.QueryContext(ctx, `
		SELECT r.id, r.url, r.host, r.title, r.summary, r.category_id, c.name, r.user_override, r.created_at, r.updated_at
		FROM resources r
		JOIN categories c ON c.id = r.category_id
		WHERE r.url ILIKE $1 OR r.title ILIKE $1 OR r.summary ILIKE $1 OR c.name ILIKE $1
		ORDER BY r.created_at DESC
		LIMIT $2
	`, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("search resources: %w", err)
	}
	defer rows.Close()

	items := make([]domain.Resource, 0)
	for rows.Next() {
		item, err := scanResource(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate search results: %w", err)
	}

	return items, nil
}

func (r *ResourceRepository) UpdateCategory(ctx context.Context, resourceID, categoryID string, userOverride bool) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE resources
		SET category_id = $1, user_override = $2, updated_at = $3
		WHERE id = $4
	`, strings.TrimSpace(categoryID), userOverride, time.Now().UTC(), strings.TrimSpace(resourceID))
	if err != nil {
		return fmt.Errorf("update resource category: %w", err)
	}

	return nil
}

func scanCategory(row interface{ Scan(dest ...any) error }) (*domain.Category, error) {
	var item domain.Category
	var source string
	if err := row.Scan(
		&item.ID,
		&item.Name,
		&item.Description,
		&source,
		&item.AcceptCount,
		&item.OverrideCount,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}

	item.Source = domain.CategorySource(source)
	return &item, nil
}

func scanResource(row interface{ Scan(dest ...any) error }) (*domain.Resource, error) {
	var item domain.Resource
	if err := row.Scan(
		&item.ID,
		&item.URL,
		&item.Host,
		&item.Title,
		&item.Summary,
		&item.CategoryID,
		&item.CategoryName,
		&item.UserOverride,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan resource: %w", err)
	}

	return &item, nil
}

type TodoRepository struct {
	db *sql.DB
}

var _ domain.TodoRepository = (*TodoRepository)(nil)

func NewTodoRepository(db *sql.DB) *TodoRepository {
	return &TodoRepository{db: db}
}

func (r *TodoRepository) Create(ctx context.Context, todo *domain.Todo) error {
	now := time.Now().UTC()
	if todo.CreatedAt.IsZero() {
		todo.CreatedAt = now
	}
	todo.UpdatedAt = now

	var dueAt any
	if todo.DueAt != nil {
		dueAt = todo.DueAt.UTC()
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO todos (id, title, details, status, due_at, resource_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, todo.ID, strings.TrimSpace(todo.Title), strings.TrimSpace(todo.Details), string(todo.Status), dueAt, todo.ResourceID, todo.CreatedAt, todo.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create todo: %w", err)
	}

	return nil
}

func (r *TodoRepository) GetByID(ctx context.Context, id string) (*domain.Todo, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, title, details, status, due_at, resource_id, created_at, updated_at
		FROM todos
		WHERE id = $1
	`, strings.TrimSpace(id))

	item, err := scanTodo(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get todo by id: %w", err)
	}

	return item, nil
}

func (r *TodoRepository) Update(ctx context.Context, todo *domain.Todo) error {
	todo.UpdatedAt = time.Now().UTC()

	var dueAt any
	if todo.DueAt != nil {
		dueAt = todo.DueAt.UTC()
	}

	_, err := r.db.ExecContext(ctx, `
		UPDATE todos
		SET title = $1, details = $2, status = $3, due_at = $4, resource_id = $5, updated_at = $6
		WHERE id = $7
	`, strings.TrimSpace(todo.Title), strings.TrimSpace(todo.Details), string(todo.Status), dueAt, todo.ResourceID, todo.UpdatedAt, strings.TrimSpace(todo.ID))
	if err != nil {
		return fmt.Errorf("update todo: %w", err)
	}

	return nil
}

func (r *TodoRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM todos
		WHERE id = $1
	`, strings.TrimSpace(id))
	if err != nil {
		return fmt.Errorf("delete todo: %w", err)
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
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list todos: %w", err)
	}
	defer rows.Close()

	items := make([]domain.Todo, 0)
	for rows.Next() {
		item, err := scanTodo(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate todos: %w", err)
	}

	return items, nil
}

type ReminderRepository struct {
	db *sql.DB
}

var _ domain.ReminderRepository = (*ReminderRepository)(nil)

func NewReminderRepository(db *sql.DB) *ReminderRepository {
	return &ReminderRepository{db: db}
}

func (r *ReminderRepository) Create(ctx context.Context, reminder *domain.Reminder) error {
	now := time.Now().UTC()
	if reminder.CreatedAt.IsZero() {
		reminder.CreatedAt = now
	}
	reminder.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO reminders (id, title, message, remind_at, status, resource_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, reminder.ID, strings.TrimSpace(reminder.Title), strings.TrimSpace(reminder.Message), reminder.RemindAt.UTC(), string(reminder.Status), reminder.ResourceID, reminder.CreatedAt, reminder.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create reminder: %w", err)
	}

	return nil
}

func (r *ReminderRepository) GetByID(ctx context.Context, id string) (*domain.Reminder, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, title, message, remind_at, status, resource_id, created_at, updated_at
		FROM reminders
		WHERE id = $1
	`, strings.TrimSpace(id))

	item, err := scanReminder(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get reminder by id: %w", err)
	}

	return item, nil
}

func (r *ReminderRepository) Update(ctx context.Context, reminder *domain.Reminder) error {
	reminder.UpdatedAt = time.Now().UTC()

	_, err := r.db.ExecContext(ctx, `
		UPDATE reminders
		SET title = $1, message = $2, remind_at = $3, status = $4, resource_id = $5, updated_at = $6
		WHERE id = $7
	`, strings.TrimSpace(reminder.Title), strings.TrimSpace(reminder.Message), reminder.RemindAt.UTC(), string(reminder.Status), reminder.ResourceID, reminder.UpdatedAt, strings.TrimSpace(reminder.ID))
	if err != nil {
		return fmt.Errorf("update reminder: %w", err)
	}

	return nil
}

func (r *ReminderRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM reminders
		WHERE id = $1
	`, strings.TrimSpace(id))
	if err != nil {
		return fmt.Errorf("delete reminder: %w", err)
	}

	return nil
}

func (r *ReminderRepository) List(ctx context.Context, limit, offset int) ([]domain.Reminder, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, message, remind_at, status, resource_id, created_at, updated_at
		FROM reminders
		ORDER BY remind_at ASC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list reminders: %w", err)
	}
	defer rows.Close()

	items := make([]domain.Reminder, 0)
	for rows.Next() {
		item, err := scanReminder(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate reminders: %w", err)
	}

	return items, nil
}

func scanTodo(row interface{ Scan(dest ...any) error }) (*domain.Todo, error) {
	var item domain.Todo
	var status string
	var dueAt sql.NullTime
	var resourceID sql.NullString
	if err := row.Scan(
		&item.ID,
		&item.Title,
		&item.Details,
		&status,
		&dueAt,
		&resourceID,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan todo: %w", err)
	}

	item.Status = domain.TodoStatus(status)
	if dueAt.Valid {
		t := dueAt.Time.UTC()
		item.DueAt = &t
	}
	if resourceID.Valid {
		id := resourceID.String
		item.ResourceID = &id
	}

	return &item, nil
}

func scanReminder(row interface{ Scan(dest ...any) error }) (*domain.Reminder, error) {
	var item domain.Reminder
	var status string
	var resourceID sql.NullString
	if err := row.Scan(
		&item.ID,
		&item.Title,
		&item.Message,
		&item.RemindAt,
		&status,
		&resourceID,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan reminder: %w", err)
	}

	item.RemindAt = item.RemindAt.UTC()
	item.Status = domain.ReminderStatus(status)
	if resourceID.Valid {
		id := resourceID.String
		item.ResourceID = &id
	}

	return &item, nil
}
