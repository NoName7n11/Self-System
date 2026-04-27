package sync

import (
	"context"
	"fmt"
	"strings"
	"time"

	"selfsystems/internal/domain"
	"selfsystems/internal/service"
)

// ReplayMutationApplier applies replayed mutations to domain services before fanout.
type ReplayMutationApplier interface {
	Apply(ctx context.Context, mutation ReplayMutation) error
}

// ReplayMutationApplierFunc adapts a function to ReplayMutationApplier.
type ReplayMutationApplierFunc func(ctx context.Context, mutation ReplayMutation) error

func (f ReplayMutationApplierFunc) Apply(ctx context.Context, mutation ReplayMutation) error {
	if f == nil {
		return nil
	}
	return f(ctx, mutation)
}

// ServiceMutationApplier routes replayed sync events into entity services.
type ServiceMutationApplier struct {
	resources  *service.ResourceService
	categories *service.CategoryService
	todos      *service.TodoService
	reminders  *service.ReminderService
}

func NewServiceMutationApplier(
	resources *service.ResourceService,
	categories *service.CategoryService,
	todos *service.TodoService,
	reminders *service.ReminderService,
) *ServiceMutationApplier {
	return &ServiceMutationApplier{
		resources:  resources,
		categories: categories,
		todos:      todos,
		reminders:  reminders,
	}
}

func (a *ServiceMutationApplier) Apply(ctx context.Context, mutation ReplayMutation) error {
	eventType := strings.TrimSpace(strings.ToLower(mutation.EventType))
	switch eventType {
	case EventTypeUpdate, EventTypeReconnected:
		return nil
	case EventTypeResourceCreated:
		return a.applyResourceCreate(ctx, mutation)
	case EventTypeResourceUpdated:
		return a.applyResourceUpdate(ctx, mutation)
	case EventTypeResourceDeleted:
		return a.applyResourceDelete(ctx, mutation)
	case EventTypeCategoryUpdated:
		return a.applyCategoryUpdate(ctx, mutation)
	case EventTypeTodoUpdated:
		return a.applyTodoUpdate(ctx, mutation)
	case EventTypeReminderUpdated:
		return a.applyReminderUpdate(ctx, mutation)
	default:
		return fmt.Errorf("unsupported replay event type %q", mutation.EventType)
	}
}

func (a *ServiceMutationApplier) applyResourceCreate(ctx context.Context, mutation ReplayMutation) error {
	if a.resources == nil {
		return fmt.Errorf("resource service is not configured")
	}

	url, _ := payloadString(mutation.Payload, "url")
	if strings.TrimSpace(url) == "" {
		return fmt.Errorf("resource create replay requires payload.url")
	}

	title, _ := payloadString(mutation.Payload, "title")
	summary, _ := payloadString(mutation.Payload, "summary")
	categoryID, _ := payloadString(mutation.Payload, "category_id")
	categoryName, _ := payloadString(mutation.Payload, "category_name", "category")

	_, err := a.resources.Create(ctx, service.CreateResourceInput{
		URL:          url,
		Title:        title,
		Summary:      summary,
		CategoryID:   categoryID,
		CategoryName: categoryName,
	})
	return err
}

func (a *ServiceMutationApplier) applyResourceUpdate(ctx context.Context, mutation ReplayMutation) error {
	if a.resources == nil {
		return fmt.Errorf("resource service is not configured")
	}

	entityID := strings.TrimSpace(firstNonEmpty(mutation.EntityID, ExtractEntityID(mutation.Payload)))
	if entityID == "" {
		return fmt.Errorf("resource replay requires entity_id")
	}

	existing, err := a.resources.GetByID(ctx, entityID)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("resource %q not found for replay update", entityID)
	}

	url := existing.URL
	if value, ok := payloadString(mutation.Payload, "url"); ok {
		url = value
	}

	title := existing.Title
	if value, ok := payloadString(mutation.Payload, "title"); ok {
		title = value
	}

	summary := existing.Summary
	if value, ok := payloadString(mutation.Payload, "summary"); ok {
		summary = value
	}

	categoryID := ""
	if value, ok := payloadString(mutation.Payload, "category_id"); ok {
		categoryID = value
	}
	categoryName := ""
	if value, ok := payloadString(mutation.Payload, "category_name", "category"); ok {
		categoryName = value
	}

	_, err = a.resources.Update(ctx, service.UpdateResourceInput{
		ID:           entityID,
		URL:          url,
		Title:        title,
		Summary:      summary,
		CategoryID:   categoryID,
		CategoryName: categoryName,
	})
	return err
}

func (a *ServiceMutationApplier) applyResourceDelete(ctx context.Context, mutation ReplayMutation) error {
	if a.resources == nil {
		return fmt.Errorf("resource service is not configured")
	}

	entityID := strings.TrimSpace(firstNonEmpty(mutation.EntityID, ExtractEntityID(mutation.Payload)))
	if entityID == "" {
		return fmt.Errorf("resource replay delete requires entity_id")
	}

	_, err := a.resources.Delete(ctx, entityID)
	return err
}

func (a *ServiceMutationApplier) applyCategoryUpdate(ctx context.Context, mutation ReplayMutation) error {
	if a.categories == nil {
		return fmt.Errorf("category service is not configured")
	}

	entityID := strings.TrimSpace(firstNonEmpty(mutation.EntityID, ExtractEntityID(mutation.Payload)))
	if deleted, _ := payloadBool(mutation.Payload, "deleted"); deleted && entityID != "" {
		_, err := a.categories.Delete(ctx, entityID)
		return err
	}

	name, nameSet := payloadString(mutation.Payload, "name")
	description, descSet := payloadString(mutation.Payload, "description")

	if entityID == "" {
		if !nameSet || strings.TrimSpace(name) == "" {
			return fmt.Errorf("category replay requires entity_id or payload.name")
		}
		_, err := a.categories.EnsureByName(ctx, name, domain.CategorySourceManual)
		return err
	}

	existing, err := a.categories.GetByID(ctx, entityID)
	if err != nil {
		return err
	}
	if existing == nil {
		if !nameSet || strings.TrimSpace(name) == "" {
			return fmt.Errorf("category %q missing and payload.name not provided", entityID)
		}
		_, err := a.categories.Create(ctx, service.CreateCategoryInput{
			Name:        name,
			Description: description,
			Source:      domain.CategorySourceManual,
		})
		return err
	}

	if !nameSet || strings.TrimSpace(name) == "" {
		name = existing.Name
	}
	if !descSet {
		description = existing.Description
	}

	_, err = a.categories.Update(ctx, service.UpdateCategoryInput{
		ID:          entityID,
		Name:        name,
		Description: description,
	})
	return err
}

func (a *ServiceMutationApplier) applyTodoUpdate(ctx context.Context, mutation ReplayMutation) error {
	if a.todos == nil {
		return fmt.Errorf("todo service is not configured")
	}

	entityID := strings.TrimSpace(firstNonEmpty(mutation.EntityID, ExtractEntityID(mutation.Payload)))
	if entityID == "" {
		return fmt.Errorf("todo replay requires entity_id")
	}
	if deleted, _ := payloadBool(mutation.Payload, "deleted"); deleted {
		_, err := a.todos.Delete(ctx, entityID)
		return err
	}

	existing, err := a.todos.GetByID(ctx, entityID)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("todo %q not found for replay update", entityID)
	}

	title := existing.Title
	if value, ok := payloadString(mutation.Payload, "title"); ok {
		title = value
	}
	details := existing.Details
	if value, ok := payloadString(mutation.Payload, "details"); ok {
		details = value
	}

	status := existing.Status
	if value, ok := payloadString(mutation.Payload, "status"); ok {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			status = domain.TodoStatus(trimmed)
		}
	}

	dueAt := existing.DueAt
	if value, ok := payloadValue(mutation.Payload, "due_at", "due"); ok {
		parsedDueAt, parseErr := payloadTimePointer(value)
		if parseErr != nil {
			return parseErr
		}
		dueAt = parsedDueAt
	}

	resourceID := existing.ResourceID
	if value, ok := payloadValue(mutation.Payload, "resource_id", "resource"); ok {
		resourceID = payloadStringPointer(value)
	}

	_, err = a.todos.Update(ctx, service.UpdateTodoInput{
		ID:         entityID,
		Title:      title,
		Details:    details,
		Status:     status,
		DueAt:      dueAt,
		ResourceID: resourceID,
	})
	return err
}

func (a *ServiceMutationApplier) applyReminderUpdate(ctx context.Context, mutation ReplayMutation) error {
	if a.reminders == nil {
		return fmt.Errorf("reminder service is not configured")
	}

	entityID := strings.TrimSpace(firstNonEmpty(mutation.EntityID, ExtractEntityID(mutation.Payload)))
	if entityID == "" {
		return fmt.Errorf("reminder replay requires entity_id")
	}
	if deleted, _ := payloadBool(mutation.Payload, "deleted"); deleted {
		_, err := a.reminders.Delete(ctx, entityID)
		return err
	}

	existing, err := a.reminders.GetByID(ctx, entityID)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("reminder %q not found for replay update", entityID)
	}

	title := existing.Title
	if value, ok := payloadString(mutation.Payload, "title"); ok {
		title = value
	}
	message := existing.Message
	if value, ok := payloadString(mutation.Payload, "message"); ok {
		message = value
	}

	status := existing.Status
	if value, ok := payloadString(mutation.Payload, "status"); ok {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			status = domain.ReminderStatus(trimmed)
		}
	}

	remindAt := existing.RemindAt
	if value, ok := payloadValue(mutation.Payload, "remind_at", "at"); ok {
		parsedRemindAt, parseErr := payloadTime(value)
		if parseErr != nil {
			return parseErr
		}
		remindAt = parsedRemindAt
	}

	resourceID := existing.ResourceID
	if value, ok := payloadValue(mutation.Payload, "resource_id", "resource"); ok {
		resourceID = payloadStringPointer(value)
	}

	_, err = a.reminders.Update(ctx, service.UpdateReminderInput{
		ID:         entityID,
		Title:      title,
		Message:    message,
		RemindAt:   remindAt,
		Status:     status,
		ResourceID: resourceID,
	})
	return err
}

func payloadValue(payload map[string]any, keys ...string) (any, bool) {
	if payload == nil {
		return nil, false
	}

	for _, key := range keys {
		value, ok := payload[key]
		if ok {
			return value, true
		}
	}
	return nil, false
}

func payloadString(payload map[string]any, keys ...string) (string, bool) {
	value, ok := payloadValue(payload, keys...)
	if !ok {
		return "", false
	}
	return strings.TrimSpace(fmt.Sprint(value)), true
}

func payloadBool(payload map[string]any, key string) (bool, bool) {
	value, ok := payloadValue(payload, key)
	if !ok {
		return false, false
	}

	switch typed := value.(type) {
	case bool:
		return typed, true
	default:
		trimmed := strings.ToLower(strings.TrimSpace(fmt.Sprint(value)))
		return trimmed == "true" || trimmed == "1" || trimmed == "yes", true
	}
}

func payloadTime(value any) (time.Time, error) {
	trimmed := strings.TrimSpace(fmt.Sprint(value))
	if trimmed == "" {
		return time.Time{}, fmt.Errorf("timestamp is required")
	}

	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return time.Time{}, fmt.Errorf("timestamp must be RFC3339")
	}
	return parsed.UTC(), nil
}

func payloadTimePointer(value any) (*time.Time, error) {
	trimmed := strings.TrimSpace(fmt.Sprint(value))
	if trimmed == "" {
		return nil, nil
	}

	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return nil, fmt.Errorf("timestamp must be RFC3339")
	}
	parsed = parsed.UTC()
	return &parsed, nil
}

func payloadStringPointer(value any) *string {
	trimmed := strings.TrimSpace(fmt.Sprint(value))
	if trimmed == "" {
		return nil
	}
	pointer := trimmed
	return &pointer
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
