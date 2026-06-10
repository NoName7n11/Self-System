package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"selfsystems/internal/domain"
)

type ResourceRepository struct {
	db *sql.DB
}

func NewResourceRepository(db *sql.DB) *ResourceRepository {
	return &ResourceRepository{db: db}
}

func (r *ResourceRepository) GetByID(ctx context.Context, id string) (*domain.Resource, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT r.id, r.url, r.host, r.title, r.summary, r.category_id, c.name,
		       r.user_override, r.extracted_data, r.save_count,
		       r.archived, r.archive_reason, r.archived_at,
		       r.created_at, r.updated_at
		FROM resources r
		JOIN categories c ON c.id = r.category_id
		WHERE r.id = ?
	`, strings.TrimSpace(id))

	resource, err := scanResource(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get resource by id: %w", err)
	}

	similarIDs, err := r.loadSimilarIDs(ctx, resource.ID)
	if err == nil {
		resource.SimilarTo = similarIDs
	}

	return resource, nil
}

func (r *ResourceRepository) FindByURL(ctx context.Context, url string) (*domain.Resource, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT r.id, r.url, r.host, r.title, r.summary, r.category_id, c.name,
		       r.user_override, r.extracted_data, r.save_count,
		       r.archived, r.archive_reason, r.archived_at,
		       r.created_at, r.updated_at
		FROM resources r
		JOIN categories c ON c.id = r.category_id
		WHERE r.url = ?
	`, strings.TrimSpace(url))

	resource, err := scanResource(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find resource by url: %w", err)
	}
	return resource, nil
}

func (r *ResourceRepository) Create(ctx context.Context, resource *domain.Resource) error {
	timestamp := nowRFC3339()
	resource.CreatedAt = parseRFC3339(timestamp)
	resource.UpdatedAt = resource.CreatedAt

	extractedJSON, err := marshalExtractedData(resource.ExtractedData)
	if err != nil {
		return fmt.Errorf("marshal extracted_data: %w", err)
	}

	saveCount := resource.SaveCount
	if saveCount < 1 {
		saveCount = 1
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO resources (id, url, host, title, summary, category_id, user_override,
		                       extracted_data, save_count, archived, archive_reason, archived_at,
		                       created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, resource.ID, strings.TrimSpace(resource.URL), strings.TrimSpace(resource.Host),
		strings.TrimSpace(resource.Title), strings.TrimSpace(resource.Summary),
		resource.CategoryID, boolToInt(resource.UserOverride),
		extractedJSON, saveCount,
		boolToInt(resource.Archived), string(resource.ArchiveReason), nullableTime(resource.ArchivedAt),
		timestamp, timestamp)
	if err != nil {
		return fmt.Errorf("create resource: %w", err)
	}
	return nil
}

func (r *ResourceRepository) Update(ctx context.Context, resource *domain.Resource) error {
	timestamp := nowRFC3339()
	resource.UpdatedAt = parseRFC3339(timestamp)

	extractedJSON, err := marshalExtractedData(resource.ExtractedData)
	if err != nil {
		return fmt.Errorf("marshal extracted_data: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
		UPDATE resources
		SET url = ?, host = ?, title = ?, summary = ?, category_id = ?, user_override = ?,
		    extracted_data = ?, updated_at = ?
		WHERE id = ?
	`, strings.TrimSpace(resource.URL), strings.TrimSpace(resource.Host),
		strings.TrimSpace(resource.Title), strings.TrimSpace(resource.Summary),
		strings.TrimSpace(resource.CategoryID), boolToInt(resource.UserOverride),
		extractedJSON, timestamp, strings.TrimSpace(resource.ID))
	if err != nil {
		return fmt.Errorf("update resource: %w", err)
	}

	return nil
}

func (r *ResourceRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM resources
		WHERE id = ?
	`, strings.TrimSpace(id))
	if err != nil {
		return fmt.Errorf("delete resource: %w", err)
	}

	return nil
}

// List returns non-archived resources ordered by created_at DESC.
func (r *ResourceRepository) List(ctx context.Context, limit, offset int) ([]domain.Resource, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT r.id, r.url, r.host, r.title, r.summary, r.category_id, c.name,
		       r.user_override, r.extracted_data, r.save_count,
		       r.archived, r.archive_reason, r.archived_at,
		       r.created_at, r.updated_at
		FROM resources r
		JOIN categories c ON c.id = r.category_id
		WHERE r.archived = 0
		ORDER BY r.created_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list resources: %w", err)
	}
	defer rows.Close()

	return scanResources(rows)
}

// ListArchived returns only archived resources ordered by archived_at DESC.
func (r *ResourceRepository) ListArchived(ctx context.Context, limit, offset int) ([]domain.Resource, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT r.id, r.url, r.host, r.title, r.summary, r.category_id, c.name,
		       r.user_override, r.extracted_data, r.save_count,
		       r.archived, r.archive_reason, r.archived_at,
		       r.created_at, r.updated_at
		FROM resources r
		JOIN categories c ON c.id = r.category_id
		WHERE r.archived = 1
		ORDER BY r.archived_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list archived resources: %w", err)
	}
	defer rows.Close()

	return scanResources(rows)
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
		SELECT r.id, r.url, r.host, r.title, r.summary, r.category_id, c.name,
		       r.user_override, r.extracted_data, r.save_count,
		       r.archived, r.archive_reason, r.archived_at,
		       r.created_at, r.updated_at
		FROM resources r
		JOIN categories c ON c.id = r.category_id
		WHERE r.archived = 0
		  AND (lower(r.url) LIKE ? OR lower(r.title) LIKE ? OR lower(r.summary) LIKE ? OR lower(c.name) LIKE ?)
		ORDER BY r.created_at DESC
		LIMIT ?
	`, pattern, pattern, pattern, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("search resources: %w", err)
	}
	defer rows.Close()

	return scanResources(rows)
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

func (r *ResourceRepository) UpdateExtractedData(ctx context.Context, resourceID string, data domain.ResourceExtractedData) error {
	extractedJSON, err := marshalExtractedData(data)
	if err != nil {
		return fmt.Errorf("marshal extracted_data: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
		UPDATE resources
		SET extracted_data = ?, updated_at = ?
		WHERE id = ?
	`, extractedJSON, nowRFC3339(), strings.TrimSpace(resourceID))
	if err != nil {
		return fmt.Errorf("update extracted_data: %w", err)
	}
	return nil
}

func (r *ResourceRepository) IncrementCounter(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE resources SET save_count = save_count + 1, updated_at = ? WHERE id = ?
	`, nowRFC3339(), strings.TrimSpace(id))
	if err != nil {
		return fmt.Errorf("increment counter: %w", err)
	}
	return nil
}

func (r *ResourceRepository) Archive(ctx context.Context, id string, reason domain.ArchiveReason) error {
	ts := nowRFC3339()
	_, err := r.db.ExecContext(ctx, `
		UPDATE resources
		SET archived = 1, archive_reason = ?, archived_at = ?, updated_at = ?
		WHERE id = ?
	`, string(reason), ts, ts, strings.TrimSpace(id))
	if err != nil {
		return fmt.Errorf("archive resource: %w", err)
	}
	return nil
}

func (r *ResourceRepository) Restore(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE resources
		SET archived = 0, archive_reason = '', archived_at = NULL, updated_at = ?
		WHERE id = ?
	`, nowRFC3339(), strings.TrimSpace(id))
	if err != nil {
		return fmt.Errorf("restore resource: %w", err)
	}
	return nil
}

func (r *ResourceRepository) BulkArchive(ctx context.Context, ids []string, reason domain.ArchiveReason) error {
	if len(ids) == 0 {
		return nil
	}
	if len(ids) > 100 {
		return fmt.Errorf("bulk archive: max 100 ids, got %d", len(ids))
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("bulk archive begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	ts := nowRFC3339()
	for _, id := range ids {
		if _, err := tx.ExecContext(ctx, `
			UPDATE resources
			SET archived = 1, archive_reason = ?, archived_at = ?, updated_at = ?
			WHERE id = ?
		`, string(reason), ts, ts, strings.TrimSpace(id)); err != nil {
			return fmt.Errorf("bulk archive resource %s: %w", id, err)
		}
	}
	return tx.Commit()
}

func (r *ResourceRepository) BulkRestore(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	if len(ids) > 100 {
		return fmt.Errorf("bulk restore: max 100 ids, got %d", len(ids))
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("bulk restore begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	ts := nowRFC3339()
	for _, id := range ids {
		if _, err := tx.ExecContext(ctx, `
			UPDATE resources
			SET archived = 0, archive_reason = '', archived_at = NULL, updated_at = ?
			WHERE id = ?
		`, ts, strings.TrimSpace(id)); err != nil {
			return fmt.Errorf("bulk restore resource %s: %w", id, err)
		}
	}
	return tx.Commit()
}

// ── similar_resources helpers ──────────────────────────────────────────────

func (r *ResourceRepository) loadSimilarIDs(ctx context.Context, resourceID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT similar_id FROM similar_resources WHERE resource_id = ?
	`, resourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ── SimilarResourceRepository ──────────────────────────────────────────────

type SimilarResourceRepository struct {
	db *sql.DB
}

func NewSimilarResourceRepository(db *sql.DB) *SimilarResourceRepository {
	return &SimilarResourceRepository{db: db}
}

func (r *SimilarResourceRepository) Upsert(ctx context.Context, resourceID, similarID string, score float64) error {
	ts := nowRFC3339()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO similar_resources (resource_id, similar_id, similarity_score, created_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(resource_id, similar_id) DO UPDATE SET similarity_score = excluded.similarity_score
	`, resourceID, similarID, score, ts)
	if err != nil {
		return fmt.Errorf("upsert similar resource: %w", err)
	}
	// Mirror the reverse direction so queries from either side work.
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO similar_resources (resource_id, similar_id, similarity_score, created_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(resource_id, similar_id) DO UPDATE SET similarity_score = excluded.similarity_score
	`, similarID, resourceID, score, ts)
	return err
}

func (r *SimilarResourceRepository) ListByResource(ctx context.Context, resourceID string) ([]domain.SimilarResource, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT resource_id, similar_id, similarity_score, created_at
		FROM similar_resources WHERE resource_id = ?
		ORDER BY similarity_score DESC
	`, resourceID)
	if err != nil {
		return nil, fmt.Errorf("list similar resources: %w", err)
	}
	defer rows.Close()

	var results []domain.SimilarResource
	for rows.Next() {
		var s domain.SimilarResource
		var createdAt string
		if err := rows.Scan(&s.ResourceID, &s.SimilarID, &s.SimilarityScore, &createdAt); err != nil {
			return nil, fmt.Errorf("scan similar resource: %w", err)
		}
		s.CreatedAt = parseRFC3339(createdAt)
		results = append(results, s)
	}
	return results, rows.Err()
}

func (r *SimilarResourceRepository) Delete(ctx context.Context, resourceID, similarID string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM similar_resources WHERE (resource_id = ? AND similar_id = ?) OR (resource_id = ? AND similar_id = ?)
	`, resourceID, similarID, similarID, resourceID)
	return err
}

// ── scan helpers ──────────────────────────────────────────────────────────

func scanResources(rows *sql.Rows) ([]domain.Resource, error) {
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

func scanResource(row interface{ Scan(dest ...any) error }) (*domain.Resource, error) {
	var resource domain.Resource
	var userOverride int
	var extractedJSON string
	var saveCount int
	var archived int
	var archiveReason string
	var archivedAt sql.NullString
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
		&extractedJSON,
		&saveCount,
		&archived,
		&archiveReason,
		&archivedAt,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan resource: %w", err)
	}

	resource.UserOverride = intToBool(userOverride)
	resource.SaveCount = saveCount
	if resource.SaveCount < 1 {
		resource.SaveCount = 1
	}
	resource.Archived = intToBool(archived)
	resource.ArchiveReason = domain.ArchiveReason(archiveReason)
	if archivedAt.Valid && archivedAt.String != "" {
		t := parseRFC3339(archivedAt.String)
		resource.ArchivedAt = &t
	}
	resource.CreatedAt = parseRFC3339(createdAt)
	resource.UpdatedAt = parseRFC3339(updatedAt)
	if err := unmarshalExtractedData(extractedJSON, &resource.ExtractedData); err != nil {
		resource.ExtractedData = domain.ResourceExtractedData{}
	}
	return &resource, nil
}

func marshalExtractedData(data domain.ResourceExtractedData) (string, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return "{}", err
	}
	return string(b), nil
}

func unmarshalExtractedData(raw string, out *domain.ResourceExtractedData) error {
	if raw == "" || raw == "{}" {
		return nil
	}
	return json.Unmarshal([]byte(raw), out)
}

// nullableTime converts *time.Time to a string for nullable columns.
func nullableTime(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}
