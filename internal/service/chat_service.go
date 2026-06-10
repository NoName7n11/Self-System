package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"selfsystems/internal/domain"
)

type ChatService struct {
	categories *CategoryService
	resources  *ResourceService
	todos      *TodoService
	reminders  *ReminderService
	graph      *GraphService
}

type ChatResult struct {
	Action   string           `json:"action"`
	Message  string           `json:"message"`
	Category *domain.Category `json:"category,omitempty"`
	Resource *domain.Resource `json:"resource,omitempty"`
	Todo     *domain.Todo     `json:"todo,omitempty"`
	Reminder *domain.Reminder `json:"reminder,omitempty"`

	Categories []domain.Category `json:"categories,omitempty"`
	Resources  []domain.Resource `json:"resources,omitempty"`
	Todos      []domain.Todo     `json:"todos,omitempty"`
	Reminders  []domain.Reminder `json:"reminders,omitempty"`
	Graph      *GraphData        `json:"graph,omitempty"`
}

func NewChatService(
	categories *CategoryService,
	resources *ResourceService,
	todos *TodoService,
	reminders *ReminderService,
	graph *GraphService,
) *ChatService {
	return &ChatService{
		categories: categories,
		resources:  resources,
		todos:      todos,
		reminders:  reminders,
		graph:      graph,
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
	case strings.HasPrefix(lower, "list categories"):
		return s.handleListCategories(ctx), nil
	case strings.HasPrefix(lower, "get category"):
		return s.handleGetCategory(ctx, strings.TrimSpace(trimmed[len("get category"):])), nil
	case strings.HasPrefix(lower, "update category"):
		return s.handleUpdateCategory(ctx, strings.TrimSpace(trimmed[len("update category"):])), nil
	case strings.HasPrefix(lower, "delete category"):
		return s.handleDeleteCategory(ctx, strings.TrimSpace(trimmed[len("delete category"):])), nil
	case strings.HasPrefix(lower, "create todo"):
		return s.handleCreateTodo(ctx, strings.TrimSpace(trimmed[len("create todo"):])), nil
	case strings.HasPrefix(lower, "todo:"):
		return s.handleCreateTodo(ctx, strings.TrimSpace(trimmed[len("todo:"):])), nil
	case strings.HasPrefix(lower, "list todos"):
		return s.handleListTodos(ctx, strings.TrimSpace(trimmed[len("list todos"):])), nil
	case strings.HasPrefix(lower, "get todo"):
		return s.handleGetTodo(ctx, strings.TrimSpace(trimmed[len("get todo"):])), nil
	case strings.HasPrefix(lower, "update todo"):
		return s.handleUpdateTodo(ctx, strings.TrimSpace(trimmed[len("update todo"):])), nil
	case strings.HasPrefix(lower, "delete todo"):
		return s.handleDeleteTodo(ctx, strings.TrimSpace(trimmed[len("delete todo"):])), nil
	case strings.HasPrefix(lower, "resource:"):
		return s.handleCreateResource(ctx, strings.TrimSpace(trimmed[len("resource:"):])), nil
	case strings.HasPrefix(lower, "save "):
		return s.handleCreateResource(ctx, strings.TrimSpace(trimmed[len("save "):])), nil
	case strings.HasPrefix(lower, "list resources"):
		return s.handleListResources(ctx, strings.TrimSpace(trimmed[len("list resources"):])), nil
	case strings.HasPrefix(lower, "get resource"):
		return s.handleGetResource(ctx, strings.TrimSpace(trimmed[len("get resource"):])), nil
	case strings.HasPrefix(lower, "update resource"):
		return s.handleUpdateResource(ctx, strings.TrimSpace(trimmed[len("update resource"):])), nil
	case strings.HasPrefix(lower, "delete resource"):
		return s.handleDeleteResource(ctx, strings.TrimSpace(trimmed[len("delete resource"):])), nil
	case strings.HasPrefix(lower, "search "):
		return s.handleSearchResources(ctx, strings.TrimSpace(trimmed[len("search "):])), nil
	case strings.HasPrefix(lower, "search:"):
		return s.handleSearchResources(ctx, strings.TrimSpace(trimmed[len("search:"):])), nil
	case strings.HasPrefix(lower, "semantic search "):
		return s.handleSemanticSearchResources(ctx, strings.TrimSpace(trimmed[len("semantic search "):])), nil
	case strings.HasPrefix(lower, "semantic:"):
		return s.handleSemanticSearchResources(ctx, strings.TrimSpace(trimmed[len("semantic:"):])), nil
	case strings.HasPrefix(lower, "create reminder"):
		return s.handleCreateReminder(ctx, strings.TrimSpace(trimmed[len("create reminder"):])), nil
	case strings.HasPrefix(lower, "reminder:"):
		return s.handleCreateReminder(ctx, strings.TrimSpace(trimmed[len("reminder:"):])), nil
	case strings.HasPrefix(lower, "list reminders"):
		return s.handleListReminders(ctx, strings.TrimSpace(trimmed[len("list reminders"):])), nil
	case strings.HasPrefix(lower, "get reminder"):
		return s.handleGetReminder(ctx, strings.TrimSpace(trimmed[len("get reminder"):])), nil
	case strings.HasPrefix(lower, "update reminder"):
		return s.handleUpdateReminder(ctx, strings.TrimSpace(trimmed[len("update reminder"):])), nil
	case strings.HasPrefix(lower, "delete reminder"):
		return s.handleDeleteReminder(ctx, strings.TrimSpace(trimmed[len("delete reminder"):])), nil
	case strings.HasPrefix(lower, "list graph"):
		return s.handleGraph(ctx, strings.TrimSpace(trimmed[len("list graph"):])), nil
	case lower == "graph":
		return s.handleGraph(ctx, ""), nil
	case strings.HasPrefix(lower, "graph:"):
		return s.handleGraph(ctx, strings.TrimSpace(trimmed[len("graph:"):])), nil
	case strings.HasPrefix(lower, "graph "):
		return s.handleGraph(ctx, strings.TrimSpace(trimmed[len("graph "):])), nil
	default:
		return ChatResult{
			Action:  "help",
			Message: "Unsupported command. Use: create/get/update/delete/list category, resource/save/get/update/delete/list, search, semantic search, create/get/update/delete/list todo, create/get/update/delete/list reminder, graph.",
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

func (s *ChatService) handleListCategories(ctx context.Context) ChatResult {
	items, err := s.categories.List(ctx)
	if err != nil {
		return ChatResult{Action: "categories_error", Message: err.Error()}
	}
	return ChatResult{Action: "categories_list", Message: "Categories loaded", Categories: items}
}

func (s *ChatService) handleGetCategory(ctx context.Context, payload string) ChatResult {
	id, err := parseCommandID(payload)
	if err != nil {
		return ChatResult{Action: "category_error", Message: err.Error()}
	}
	if id == "" {
		return ChatResult{Action: "category_error", Message: "category id is required"}
	}

	category, err := s.categories.GetByID(ctx, id)
	if err != nil {
		return ChatResult{Action: "category_error", Message: err.Error()}
	}
	if category == nil {
		return ChatResult{Action: "category_error", Message: "category not found"}
	}

	return ChatResult{Action: "category_retrieved", Message: "Category loaded", Category: category}
}

func (s *ChatService) handleUpdateCategory(ctx context.Context, payload string) ChatResult {
	parts := applyAllowlist(parsePipePayload(payload), "id", "name", "description")
	id := strings.TrimSpace(parts["id"])
	if id == "" {
		fallback, err := parseCommandID(parts["value"])
		if err != nil {
			return ChatResult{Action: "category_error", Message: err.Error()}
		}
		id = fallback
	}
	if id == "" {
		return ChatResult{Action: "category_error", Message: "category id is required"}
	}

	existing, err := s.categories.GetByID(ctx, id)
	if err != nil {
		return ChatResult{Action: "category_error", Message: err.Error()}
	}
	if existing == nil {
		return ChatResult{Action: "category_error", Message: "category not found"}
	}

	name := existing.Name
	if value, ok := parts["name"]; ok {
		name = strings.TrimSpace(value)
	}

	description := existing.Description
	if value, ok := parts["description"]; ok {
		description = strings.TrimSpace(value)
	}

	updated, err := s.categories.Update(ctx, UpdateCategoryInput{
		ID:          id,
		Name:        name,
		Description: description,
	})
	if err != nil {
		return ChatResult{Action: "category_error", Message: err.Error()}
	}
	if updated == nil {
		return ChatResult{Action: "category_error", Message: "category not found"}
	}

	return ChatResult{Action: "category_updated", Message: "Category updated", Category: updated}
}

func (s *ChatService) handleDeleteCategory(ctx context.Context, payload string) ChatResult {
	id, err := parseCommandID(payload)
	if err != nil {
		return ChatResult{Action: "category_error", Message: err.Error()}
	}
	if id == "" {
		return ChatResult{Action: "category_error", Message: "category id is required"}
	}

	deleted, err := s.categories.Delete(ctx, id)
	if err != nil {
		return ChatResult{Action: "category_error", Message: err.Error()}
	}
	if !deleted {
		return ChatResult{Action: "category_error", Message: "category not found"}
	}

	return ChatResult{Action: "category_deleted", Message: "Category deleted"}
}

func (s *ChatService) handleCreateTodo(ctx context.Context, payload string) ChatResult {
	parts := applyAllowlist(parsePipePayload(payload), "title", "due", "resource", "details")
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

func (s *ChatService) handleListTodos(ctx context.Context, payload string) ChatResult {
	parts := applyAllowlist(parsePipePayload(payload), "limit")
	limit := parseLimit(parts["limit"], 20)
	items, err := s.todos.List(ctx, limit, 0)
	if err != nil {
		return ChatResult{Action: "todos_error", Message: err.Error()}
	}
	return ChatResult{Action: "todos_list", Message: "Todos loaded", Todos: items}
}

func (s *ChatService) handleGetTodo(ctx context.Context, payload string) ChatResult {
	id, err := parseCommandID(payload)
	if err != nil {
		return ChatResult{Action: "todo_error", Message: err.Error()}
	}
	if id == "" {
		return ChatResult{Action: "todo_error", Message: "todo id is required"}
	}

	todo, err := s.todos.GetByID(ctx, id)
	if err != nil {
		return ChatResult{Action: "todo_error", Message: err.Error()}
	}
	if todo == nil {
		return ChatResult{Action: "todo_error", Message: "todo not found"}
	}

	return ChatResult{Action: "todo_retrieved", Message: "Todo loaded", Todo: todo}
}

func (s *ChatService) handleUpdateTodo(ctx context.Context, payload string) ChatResult {
	parts := applyAllowlist(parsePipePayload(payload), "id", "title", "details", "status", "due", "due_at", "resource")
	id := strings.TrimSpace(parts["id"])
	if id == "" {
		fallback, err := parseCommandID(parts["value"])
		if err != nil {
			return ChatResult{Action: "todo_error", Message: err.Error()}
		}
		id = fallback
	}
	if id == "" {
		return ChatResult{Action: "todo_error", Message: "todo id is required"}
	}

	existing, err := s.todos.GetByID(ctx, id)
	if err != nil {
		return ChatResult{Action: "todo_error", Message: err.Error()}
	}
	if existing == nil {
		return ChatResult{Action: "todo_error", Message: "todo not found"}
	}

	title := existing.Title
	if value, ok := parts["title"]; ok {
		title = strings.TrimSpace(value)
	}

	details := existing.Details
	if value, ok := parts["details"]; ok {
		details = strings.TrimSpace(value)
	}

	status := existing.Status
	if value, ok := parts["status"]; ok && strings.TrimSpace(value) != "" {
		status = domain.TodoStatus(strings.TrimSpace(value))
	}

	dueAt := existing.DueAt
	if value, ok := parts["due"]; ok {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			dueAt = nil
		} else {
			parsed, parseErr := time.Parse(time.RFC3339, trimmed)
			if parseErr != nil {
				return ChatResult{Action: "todo_error", Message: "due must be RFC3339 timestamp"}
			}
			dueAt = &parsed
		}
	} else if value, ok := parts["due_at"]; ok {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			dueAt = nil
		} else {
			parsed, parseErr := time.Parse(time.RFC3339, trimmed)
			if parseErr != nil {
				return ChatResult{Action: "todo_error", Message: "due must be RFC3339 timestamp"}
			}
			dueAt = &parsed
		}
	}

	resourceID := existing.ResourceID
	if value, ok := parts["resource"]; ok {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			resourceID = nil
		} else {
			temp := trimmed
			resourceID = &temp
		}
	}

	updated, err := s.todos.Update(ctx, UpdateTodoInput{
		ID:         id,
		Title:      title,
		Details:    details,
		Status:     status,
		DueAt:      dueAt,
		ResourceID: resourceID,
	})
	if err != nil {
		return ChatResult{Action: "todo_error", Message: err.Error()}
	}
	if updated == nil {
		return ChatResult{Action: "todo_error", Message: "todo not found"}
	}

	return ChatResult{Action: "todo_updated", Message: "Todo updated", Todo: updated}
}

func (s *ChatService) handleDeleteTodo(ctx context.Context, payload string) ChatResult {
	id, err := parseCommandID(payload)
	if err != nil {
		return ChatResult{Action: "todo_error", Message: err.Error()}
	}
	if id == "" {
		return ChatResult{Action: "todo_error", Message: "todo id is required"}
	}

	deleted, err := s.todos.Delete(ctx, id)
	if err != nil {
		return ChatResult{Action: "todo_error", Message: err.Error()}
	}
	if !deleted {
		return ChatResult{Action: "todo_error", Message: "todo not found"}
	}

	return ChatResult{Action: "todo_deleted", Message: "Todo deleted"}
}

func (s *ChatService) handleCreateResource(ctx context.Context, payload string) ChatResult {
	parts := applyAllowlist(parsePipePayload(payload), "title", "summary", "category")
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

func (s *ChatService) handleListResources(ctx context.Context, payload string) ChatResult {
	parts := applyAllowlist(parsePipePayload(payload), "limit")
	limit := parseLimit(parts["limit"], 20)
	items, err := s.resources.List(ctx, limit, 0)
	if err != nil {
		return ChatResult{Action: "resources_error", Message: err.Error()}
	}
	return ChatResult{Action: "resources_list", Message: "Resources loaded", Resources: items}
}

func (s *ChatService) handleGetResource(ctx context.Context, payload string) ChatResult {
	id, err := parseCommandID(payload)
	if err != nil {
		return ChatResult{Action: "resource_error", Message: err.Error()}
	}
	if id == "" {
		return ChatResult{Action: "resource_error", Message: "resource id is required"}
	}

	resource, err := s.resources.GetByID(ctx, id)
	if err != nil {
		return ChatResult{Action: "resource_error", Message: err.Error()}
	}
	if resource == nil {
		return ChatResult{Action: "resource_error", Message: "resource not found"}
	}

	return ChatResult{Action: "resource_retrieved", Message: "Resource loaded", Resource: resource}
}

func (s *ChatService) handleUpdateResource(ctx context.Context, payload string) ChatResult {
	parts := applyAllowlist(parsePipePayload(payload), "id", "url", "title", "summary", "category_id", "category")
	id := strings.TrimSpace(parts["id"])
	if id == "" {
		fallback, err := parseCommandID(parts["value"])
		if err != nil {
			return ChatResult{Action: "resource_error", Message: err.Error()}
		}
		id = fallback
	}
	if id == "" {
		return ChatResult{Action: "resource_error", Message: "resource id is required"}
	}

	existing, err := s.resources.GetByID(ctx, id)
	if err != nil {
		return ChatResult{Action: "resource_error", Message: err.Error()}
	}
	if existing == nil {
		return ChatResult{Action: "resource_error", Message: "resource not found"}
	}

	url := existing.URL
	if value, ok := parts["url"]; ok {
		url = strings.TrimSpace(value)
	}

	title := existing.Title
	if value, ok := parts["title"]; ok {
		title = strings.TrimSpace(value)
	}

	summary := existing.Summary
	if value, ok := parts["summary"]; ok {
		summary = strings.TrimSpace(value)
	}

	updated, err := s.resources.Update(ctx, UpdateResourceInput{
		ID:           id,
		URL:          url,
		Title:        title,
		Summary:      summary,
		CategoryID:   strings.TrimSpace(parts["category_id"]),
		CategoryName: strings.TrimSpace(parts["category"]),
	})
	if err != nil {
		return ChatResult{Action: "resource_error", Message: err.Error()}
	}
	if updated == nil {
		return ChatResult{Action: "resource_error", Message: "resource not found"}
	}

	return ChatResult{Action: "resource_updated", Message: "Resource updated", Resource: updated}
}

func (s *ChatService) handleDeleteResource(ctx context.Context, payload string) ChatResult {
	id, err := parseCommandID(payload)
	if err != nil {
		return ChatResult{Action: "resource_error", Message: err.Error()}
	}
	if id == "" {
		return ChatResult{Action: "resource_error", Message: "resource id is required"}
	}

	deleted, err := s.resources.Delete(ctx, id)
	if err != nil {
		return ChatResult{Action: "resource_error", Message: err.Error()}
	}
	if !deleted {
		return ChatResult{Action: "resource_error", Message: "resource not found"}
	}

	return ChatResult{Action: "resource_deleted", Message: "Resource deleted"}
}

func (s *ChatService) handleSearchResources(ctx context.Context, payload string) ChatResult {
	parts := applyAllowlist(parsePipePayload(payload), "query", "q", "limit")
	query := firstNonEmpty(parts["query"], parts["q"], parts["value"])
	if strings.TrimSpace(query) == "" {
		return ChatResult{Action: "search_error", Message: "query is required"}
	}
	limit := parseLimit(parts["limit"], 10)
	items, err := s.resources.Search(ctx, query, limit)
	if err != nil {
		return ChatResult{Action: "search_error", Message: err.Error()}
	}
	return ChatResult{Action: "resources_search", Message: "Search complete", Resources: items}
}

func (s *ChatService) handleSemanticSearchResources(ctx context.Context, payload string) ChatResult {
	parts := applyAllowlist(parsePipePayload(payload), "query", "q", "limit")
	query := firstNonEmpty(parts["query"], parts["q"], parts["value"])
	if strings.TrimSpace(query) == "" {
		return ChatResult{Action: "semantic_search_error", Message: "query is required"}
	}
	limit := parseLimit(parts["limit"], 10)
	items, err := s.resources.SemanticSearch(ctx, query, limit)
	if err != nil {
		return ChatResult{Action: "semantic_search_error", Message: err.Error()}
	}
	return ChatResult{Action: "resources_semantic_search", Message: "Semantic search complete", Resources: items}
}

func (s *ChatService) handleCreateReminder(ctx context.Context, payload string) ChatResult {
	parts := applyAllowlist(parsePipePayload(payload), "at", "title", "message", "resource")
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

func (s *ChatService) handleListReminders(ctx context.Context, payload string) ChatResult {
	parts := applyAllowlist(parsePipePayload(payload), "limit")
	limit := parseLimit(parts["limit"], 20)
	items, err := s.reminders.List(ctx, limit, 0)
	if err != nil {
		return ChatResult{Action: "reminders_error", Message: err.Error()}
	}
	return ChatResult{Action: "reminders_list", Message: "Reminders loaded", Reminders: items}
}

func (s *ChatService) handleGetReminder(ctx context.Context, payload string) ChatResult {
	id, err := parseCommandID(payload)
	if err != nil {
		return ChatResult{Action: "reminder_error", Message: err.Error()}
	}
	if id == "" {
		return ChatResult{Action: "reminder_error", Message: "reminder id is required"}
	}

	reminder, err := s.reminders.GetByID(ctx, id)
	if err != nil {
		return ChatResult{Action: "reminder_error", Message: err.Error()}
	}
	if reminder == nil {
		return ChatResult{Action: "reminder_error", Message: "reminder not found"}
	}

	return ChatResult{Action: "reminder_retrieved", Message: "Reminder loaded", Reminder: reminder}
}

func (s *ChatService) handleUpdateReminder(ctx context.Context, payload string) ChatResult {
	parts := applyAllowlist(parsePipePayload(payload), "id", "title", "message", "status", "at", "remind_at", "resource")
	id := strings.TrimSpace(parts["id"])
	if id == "" {
		fallback, err := parseCommandID(parts["value"])
		if err != nil {
			return ChatResult{Action: "reminder_error", Message: err.Error()}
		}
		id = fallback
	}
	if id == "" {
		return ChatResult{Action: "reminder_error", Message: "reminder id is required"}
	}

	existing, err := s.reminders.GetByID(ctx, id)
	if err != nil {
		return ChatResult{Action: "reminder_error", Message: err.Error()}
	}
	if existing == nil {
		return ChatResult{Action: "reminder_error", Message: "reminder not found"}
	}

	title := existing.Title
	if value, ok := parts["title"]; ok {
		title = strings.TrimSpace(value)
	}

	message := existing.Message
	if value, ok := parts["message"]; ok {
		message = strings.TrimSpace(value)
	}

	status := existing.Status
	if value, ok := parts["status"]; ok && strings.TrimSpace(value) != "" {
		status = domain.ReminderStatus(strings.TrimSpace(value))
	}

	remindAt := existing.RemindAt
	if value, ok := parts["at"]; ok {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return ChatResult{Action: "reminder_error", Message: "at=<RFC3339> is required"}
		}
		parsed, parseErr := time.Parse(time.RFC3339, trimmed)
		if parseErr != nil {
			return ChatResult{Action: "reminder_error", Message: "invalid timestamp format"}
		}
		remindAt = parsed
	} else if value, ok := parts["remind_at"]; ok {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return ChatResult{Action: "reminder_error", Message: "remind_at is required"}
		}
		parsed, parseErr := time.Parse(time.RFC3339, trimmed)
		if parseErr != nil {
			return ChatResult{Action: "reminder_error", Message: "invalid timestamp format"}
		}
		remindAt = parsed
	}

	resourceID := existing.ResourceID
	if value, ok := parts["resource"]; ok {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			resourceID = nil
		} else {
			temp := trimmed
			resourceID = &temp
		}
	}

	updated, err := s.reminders.Update(ctx, UpdateReminderInput{
		ID:         id,
		Title:      title,
		Message:    message,
		RemindAt:   remindAt,
		Status:     status,
		ResourceID: resourceID,
	})
	if err != nil {
		return ChatResult{Action: "reminder_error", Message: err.Error()}
	}
	if updated == nil {
		return ChatResult{Action: "reminder_error", Message: "reminder not found"}
	}

	return ChatResult{Action: "reminder_updated", Message: "Reminder updated", Reminder: updated}
}

func (s *ChatService) handleDeleteReminder(ctx context.Context, payload string) ChatResult {
	id, err := parseCommandID(payload)
	if err != nil {
		return ChatResult{Action: "reminder_error", Message: err.Error()}
	}
	if id == "" {
		return ChatResult{Action: "reminder_error", Message: "reminder id is required"}
	}

	deleted, err := s.reminders.Delete(ctx, id)
	if err != nil {
		return ChatResult{Action: "reminder_error", Message: err.Error()}
	}
	if !deleted {
		return ChatResult{Action: "reminder_error", Message: "reminder not found"}
	}

	return ChatResult{Action: "reminder_deleted", Message: "Reminder deleted"}
}

func (s *ChatService) handleGraph(ctx context.Context, payload string) ChatResult {
	if s.graph == nil {
		return ChatResult{Action: "graph_error", Message: "graph service is not configured"}
	}

	parts := applyAllowlist(parsePipePayload(payload), "limit")
	limit := parseGraphLimit(parts["limit"], 1000)
	graph, err := s.graph.Build(ctx, limit)
	if err != nil {
		return ChatResult{Action: "graph_error", Message: err.Error()}
	}

	return ChatResult{Action: "graph_data", Message: "Graph loaded", Graph: &graph}
}

func splitTwo(input string) (string, string) {
	parts := strings.SplitN(strings.TrimSpace(input), "|", 2)
	if len(parts) == 1 {
		return strings.TrimSpace(parts[0]), ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

func parseCommandID(input string) (string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", nil
	}

	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return "", nil
	}
	if len(parts) > 1 {
		return "", fmt.Errorf("expected a single id token, got %d", len(parts))
	}

	return strings.TrimSpace(parts[0]), nil
}

func applyAllowlist(parts map[string]string, allowed ...string) map[string]string {
	if parts == nil {
		return parts
	}
	keep := map[string]struct{}{
		"value": {},
		"url":   {},
	}
	for _, key := range allowed {
		keep[strings.ToLower(strings.TrimSpace(key))] = struct{}{}
	}
	for k := range parts {
		if _, ok := keep[k]; !ok {
			delete(parts, k)
		}
	}
	return parts
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

func parseLimit(raw string, fallback int) int {
	value := strings.TrimSpace(raw)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	if parsed > 100 {
		return 100
	}
	return parsed
}

func parseGraphLimit(raw string, fallback int) int {
	value := strings.TrimSpace(raw)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}

	if parsed > 5000 {
		return 5000
	}

	return parsed
}
