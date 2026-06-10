package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"selfsystems/internal/domain"
)

type ReminderRepository struct {
	db *sql.DB
}

func NewReminderRepository(db *sql.DB) *ReminderRepository {
	return &ReminderRepository{db: db}
}

func (r *ReminderRepository) GetByID(ctx context.Context, id string) (*domain.Reminder, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, title, message, remind_at, status, resource_id, created_at, updated_at
		FROM reminders
		WHERE id = ?
	`, strings.TrimSpace(id))

	reminder, err := scanReminder(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get reminder by id: %w", err)
	}

	return reminder, nil
}

func (r *ReminderRepository) Create(ctx context.Context, reminder *domain.Reminder) error {
	timestamp := nowRFC3339()
	reminder.CreatedAt = parseRFC3339(timestamp)
	reminder.UpdatedAt = reminder.CreatedAt

	remindAt := reminder.RemindAt.UTC().Format(timeLayout)

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO reminders (id, title, message, remind_at, status, resource_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, reminder.ID, reminder.Title, reminder.Message, remindAt, string(reminder.Status), reminder.ResourceID, timestamp, timestamp)
	if err != nil {
		return fmt.Errorf("create reminder: %w", err)
	}

	return nil
}

func (r *ReminderRepository) Update(ctx context.Context, reminder *domain.Reminder) error {
	timestamp := nowRFC3339()
	reminder.UpdatedAt = parseRFC3339(timestamp)

	remindAt := reminder.RemindAt.UTC().Format(timeLayout)

	_, err := r.db.ExecContext(ctx, `
		UPDATE reminders
		SET title = ?, message = ?, remind_at = ?, status = ?, resource_id = ?, updated_at = ?
		WHERE id = ?
	`, strings.TrimSpace(reminder.Title), strings.TrimSpace(reminder.Message), remindAt, string(reminder.Status), reminder.ResourceID, timestamp, strings.TrimSpace(reminder.ID))
	if err != nil {
		return fmt.Errorf("update reminder: %w", err)
	}

	return nil
}

func (r *ReminderRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM reminders
		WHERE id = ?
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
		LIMIT ? OFFSET ?
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

func scanReminder(row interface{ Scan(dest ...any) error }) (*domain.Reminder, error) {
	var item domain.Reminder
	var remindAt string
	var status string
	var resourceID sql.NullString
	var createdAt string
	var updatedAt string
	if err := row.Scan(
		&item.ID,
		&item.Title,
		&item.Message,
		&remindAt,
		&status,
		&resourceID,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan reminder: %w", err)
	}

	item.RemindAt = parseRFC3339(remindAt)
	item.Status = domain.ReminderStatus(status)
	if resourceID.Valid {
		id := resourceID.String
		item.ResourceID = &id
	}
	item.CreatedAt = parseRFC3339(createdAt)
	item.UpdatedAt = parseRFC3339(updatedAt)

	return &item, nil
}
