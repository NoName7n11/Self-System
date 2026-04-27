package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"selfsystems/internal/domain"
)

type CategoryRepository struct {
	db *sql.DB
}

func NewCategoryRepository(db *sql.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) List(ctx context.Context) ([]domain.Category, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, description, source, accept_count, override_count, created_at, updated_at
		FROM categories
		ORDER BY lower(name) ASC`)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	categories := make([]domain.Category, 0)
	for rows.Next() {
		var c domain.Category
		var source string
		var createdAt string
		var updatedAt string
		if err := rows.Scan(
			&c.ID,
			&c.Name,
			&c.Description,
			&source,
			&c.AcceptCount,
			&c.OverrideCount,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		c.Source = domain.CategorySource(source)
		c.CreatedAt = parseRFC3339(createdAt)
		c.UpdatedAt = parseRFC3339(updatedAt)
		categories = append(categories, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate categories: %w", err)
	}

	return categories, nil
}

func (r *CategoryRepository) GetByID(ctx context.Context, id string) (*domain.Category, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, description, source, accept_count, override_count, created_at, updated_at
		FROM categories
		WHERE id = ?`, id)

	category, err := scanCategory(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get category by id: %w", err)
	}

	return category, nil
}

func (r *CategoryRepository) GetByName(ctx context.Context, name string) (*domain.Category, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, description, source, accept_count, override_count, created_at, updated_at
		FROM categories
		WHERE lower(name) = lower(?)`, strings.TrimSpace(name))

	category, err := scanCategory(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get category by name: %w", err)
	}

	return category, nil
}

func (r *CategoryRepository) Create(ctx context.Context, c *domain.Category) error {
	timestamp := nowRFC3339()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = parseRFC3339(timestamp)
	}
	c.UpdatedAt = parseRFC3339(timestamp)

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO categories (id, name, description, source, accept_count, override_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, c.ID, strings.TrimSpace(c.Name), strings.TrimSpace(c.Description), string(c.Source), c.AcceptCount, c.OverrideCount, timestamp, timestamp)
	if err != nil {
		return fmt.Errorf("create category: %w", err)
	}
	return nil
}

func (r *CategoryRepository) Update(ctx context.Context, c *domain.Category) error {
	timestamp := nowRFC3339()
	c.UpdatedAt = parseRFC3339(timestamp)

	_, err := r.db.ExecContext(ctx, `
		UPDATE categories
		SET name = ?, description = ?, source = ?, updated_at = ?
		WHERE id = ?
	`, strings.TrimSpace(c.Name), strings.TrimSpace(c.Description), string(c.Source), timestamp, strings.TrimSpace(c.ID))
	if err != nil {
		return fmt.Errorf("update category: %w", err)
	}

	return nil
}

func (r *CategoryRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM categories
		WHERE id = ?
	`, strings.TrimSpace(id))
	if err != nil {
		return fmt.Errorf("delete category: %w", err)
	}

	return nil
}

func (r *CategoryRepository) IncrementAccept(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE categories
		SET accept_count = accept_count + 1, updated_at = ?
		WHERE id = ?
	`, nowRFC3339(), id)
	if err != nil {
		return fmt.Errorf("increment category accept: %w", err)
	}
	return nil
}

func (r *CategoryRepository) IncrementOverride(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE categories
		SET override_count = override_count + 1, updated_at = ?
		WHERE id = ?
	`, nowRFC3339(), id)
	if err != nil {
		return fmt.Errorf("increment category override: %w", err)
	}
	return nil
}

func scanCategory(row interface{ Scan(dest ...any) error }) (*domain.Category, error) {
	var c domain.Category
	var source string
	var createdAt string
	var updatedAt string
	if err := row.Scan(
		&c.ID,
		&c.Name,
		&c.Description,
		&source,
		&c.AcceptCount,
		&c.OverrideCount,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, err
	}
	c.Source = domain.CategorySource(source)
	c.CreatedAt = parseRFC3339(createdAt)
	c.UpdatedAt = parseRFC3339(updatedAt)
	return &c, nil
}
