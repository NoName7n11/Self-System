package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"selfsystems/internal/domain"
)

type ChatService struct {
	categories *CategoryService
	resources  *ResourceService
	todos      *TodoService
	reminders  *ReminderService
}

type ChatResult struct {
	Action   string            `json:"action"`
	Message  string            `json:"message"`
	Category *domain.Category  `json:"category,omitempty"`
	Resource *domain.Resource  `json:"resource,omitempty"`
	Todo     *domain.Todo      `json:"todo,omitempty"`
	Reminder *domain.Reminder  `json:"reminder,omitempty"`
}

func NewChatService(
	categories *CategoryService,
	resources *ResourceService,
	todos *TodoService,
	reminders *ReminderService,
) *ChatService {
	return &ChatService{
		categories: categories,
		resources:  resources,
		todos:      todos,
		reminders:  reminders,
	}
}

func (s *ChatService) Execute(ctx context.Context, message string) (ChatResult, error) {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return ChatResult{}, fmt.Errorf("message is required")
	}

	lower := strings.ToLower(trimmed)

	switch {
	case strings.HasPrefix(lower, "create category"):
		return s.handleCreateCategory(ctx, strings.TrimSpace(trimmed[len("create category"):])), nil
	case strings.HasPrefix(lower, "category:"):
		return s.handleCreateCategory(ctx, strings.TrimSpace(trimmed[len("category:"):])), nil
	case strings.HasPrefix(lower, "create todo"):
		return s.handleCreateTodo(ctx, strings.TrimSpace(trimmed[len("create todo"):])), nil
	case strings.HasPrefix(lower, "todo:"):
		return s.handleCreateTodo(ctx, strings.TrimSpace(trimmed[len("todo:"):])), nil
	case strings.HasPrefix(lower, "resource:"):
		return s.handleCreateResource(ctx, strings.TrimSpace(trimmed[len("resource:"):])), nil
	case strings.HasPrefix(lower, "save "):
		return s.handleCreateResource(ctx, strings.TrimSpace(trimmed[len("save "):])), nil
	case strings.HasPrefix(lower, "create reminder"):
		return s.handleCreateReminder(ctx, strings.TrimSpace(trimmed[len("create reminder"):])), nil
	case strings.HasPrefix(lower, "reminder:"):
		return s.handleCreateReminder(ctx, strings.TrimSpace(trimmed[len("reminder:"):])), nil
	default:
		return ChatResult{
			Action:  "help",
			Message: "Unsupported command. Use: create category, create todo, resource:, or reminder:",
		}, nil
	}
}

func (s *ChatService) handleCreateCategory(ctx context.Context, payload string) ChatResult {
	name, description := splitTwo(payload)
	category, err := s.categories.Create(ctx, CreateCategoryInput{
		Name:        name,
		Description: description,
		Source:      domain.CategorySourceManual,
	})
	if err != nil {
		return ChatResult{Action: "category_error", Message: err.Error()}
	}
	return ChatResult{Action: "category_created", Message: "Category created", Category: &category}
}

func (s *ChatService) handleCreateTodo(ctx context.Context, payload string) ChatResult {
	parts := parsePipePayload(payload)
	title := parts["value"]
	if customTitle := parts["title"]; customTitle != "" {
		title = customTitle
	}

	var dueAt *time.Time
	if rawDue := parts["due"]; rawDue != "" {
		parsed, err := time.Parse(time.RFC3339, rawDue)
		if err != nil {
			return ChatResult{Action: "todo_error", Message: "due must be RFC3339 timestamp"}
		}
		dueAt = &parsed
	}

	var resourceID *string
	if value := parts["resource"]; value != "" {
		temp := value
		resourceID = &temp
	}

	todo, err := s.todos.Create(ctx, CreateTodoInput{
		Title:      title,
		Details:    parts["details"],
		DueAt:      dueAt,
		ResourceID: resourceID,
	})
	if err != nil {
		return ChatResult{Action: "todo_error", Message: err.Error()}
	}

	return ChatResult{Action: "todo_created", Message: "Todo created", Todo: &todo}
}

func (s *ChatService) handleCreateResource(ctx context.Context, payload string) ChatResult {
	parts := parsePipePayload(payload)
	resource, err := s.resources.Create(ctx, CreateResourceInput{
		URL:          firstNonEmpty(parts["url"], parts["value"]),
		Title:        parts["title"],
		Summary:      parts["summary"],
		CategoryName: parts["category"],
	})
	if err != nil {
		return ChatResult{Action: "resource_error", Message: err.Error()}
	}
	return ChatResult{Action: "resource_created", Message: "Resource saved", Resource: &resource}
}

func (s *ChatService) handleCreateReminder(ctx context.Context, payload string) ChatResult {
	parts := parsePipePayload(payload)
	atRaw := parts["at"]
	if atRaw == "" {
		return ChatResult{Action: "reminder_error", Message: "at=<RFC3339> is required"}
	}
	at, err := time.Parse(time.RFC3339, atRaw)
	if err != nil {
		return ChatResult{Action: "reminder_error", Message: "invalid timestamp format"}
	}

	var resourceID *string
	if value := parts["resource"]; value != "" {
		temp := value
		resourceID = &temp
	}

	reminder, err := s.reminders.Create(ctx, CreateReminderInput{
		Title:      firstNonEmpty(parts["title"], parts["value"]),
		Message:    parts["message"],
		RemindAt:   at,
		ResourceID: resourceID,
	})
	if err != nil {
		return ChatResult{Action: "reminder_error", Message: err.Error()}
	}

	return ChatResult{Action: "reminder_created", Message: "Reminder created", Reminder: &reminder}
}

func splitTwo(input string) (string, string) {
	parts := strings.SplitN(strings.TrimSpace(input), "|", 2)
	if len(parts) == 1 {
		return strings.TrimSpace(parts[0]), ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

func parsePipePayload(input string) map[string]string {
	result := map[string]string{}
	segments := strings.Split(input, "|")
	if len(segments) == 0 {
		return result
	}
	result["value"] = strings.TrimSpace(segments[0])

	for _, segment := range segments[1:] {
		part := strings.TrimSpace(segment)
		if part == "" {
			continue
		}
		pieces := strings.SplitN(part, "=", 2)
		if len(pieces) == 2 {
			result[strings.ToLower(strings.TrimSpace(pieces[0]))] = strings.TrimSpace(pieces[1])
		}
	}

	if result["url"] == "" && strings.HasPrefix(strings.ToLower(result["value"]), "http") {
		result["url"] = result["value"]
	}

	return result
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
