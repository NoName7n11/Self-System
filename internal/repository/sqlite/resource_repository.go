package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"selfsystems/internal/domain"
)

type ResourceRepository struct {
	db *sql.DB
}

func NewResourceRepository(db *sql.DB) *ResourceRepository {
	return &ResourceRepository{db: db}
}

func (r *ResourceRepository) Create(ctx context.Context, resource *domain.Resource) error {
	timestamp := nowRFC3339()
	resource.CreatedAt = parseRFC3339(timestamp)
	resource.UpdatedAt = resource.CreatedAt

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO resources (id, url, host, title, summary, category_id, user_override, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, resource.ID, strings.TrimSpace(resource.URL), strings.TrimSpace(resource.Host), strings.TrimSpace(resource.Title), strings.TrimSpace(resource.Summary), resource.CategoryID, boolToInt(resource.UserOverride), timestamp, timestamp)
	if err != nil {
		return fmt.Errorf("create resource: %w", err)
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
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list resources: %w", err)
	}
	defer rows.Close()

	items := make([]domain.Resource, 0)
	for rows.Next() {
		resource, err := scanResource(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *resource)
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
	pattern := "%" + strings.ToLower(query) + "%"

	rows, err := r.db.QueryContext(ctx, `
		SELECT r.id, r.url, r.host, r.title, r.summary, r.category_id, c.name, r.user_override, r.created_at, r.updated_at
		FROM resources r
		JOIN categories c ON c.id = r.category_id
		WHERE lower(r.url) LIKE ? OR lower(r.title) LIKE ? OR lower(r.summary) LIKE ? OR lower(c.name) LIKE ?
		ORDER BY r.created_at DESC
		LIMIT ?
	`, pattern, pattern, pattern, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("search resources: %w", err)
	}
	defer rows.Close()

	items := make([]domain.Resource, 0)
	for rows.Next() {
		resource, err := scanResource(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *resource)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate search results: %w", err)
	}

	return items, nil
}

func (r *ResourceRepository) UpdateCategory(ctx context.Context, resourceID, categoryID string, userOverride bool) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE resources
		SET category_id = ?, user_override = ?, updated_at = ?
		WHERE id = ?
	`, categoryID, boolToInt(userOverride), nowRFC3339(), resourceID)
	if err != nil {
		return fmt.Errorf("update resource category: %w", err)
	}
	return nil
}

func scanResource(row interface{ Scan(dest ...any) error }) (*domain.Resource, error) {
	var resource domain.Resource
	var userOverride int
	var createdAt string
	var updatedAt string
	if err := row.Scan(
		&resource.ID,
		&resource.URL,
		&resource.Host,
		&resource.Title,
		&resource.Summary,
		&resource.CategoryID,
		&resource.CategoryName,
		&userOverride,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan resource: %w", err)
	}
	resource.UserOverride = intToBool(userOverride)
	resource.CreatedAt = parseRFC3339(createdAt)
	resource.UpdatedAt = parseRFC3339(updatedAt)
	return &resource, nil
}
