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
	case strings.HasPrefix(lower, "create todo"):
		return s.handleCreateTodo(ctx, strings.TrimSpace(trimmed[len("create todo"):])), nil
	case strings.HasPrefix(lower, "todo:"):
		return s.handleCreateTodo(ctx, strings.TrimSpace(trimmed[len("todo:"):])), nil
	case strings.HasPrefix(lower, "list todos"):
		return s.handleListTodos(ctx, strings.TrimSpace(trimmed[len("list todos"):])), nil
	case strings.HasPrefix(lower, "resource:"):
		return s.handleCreateResource(ctx, strings.TrimSpace(trimmed[len("resource:"):])), nil
	case strings.HasPrefix(lower, "save "):
		return s.handleCreateResource(ctx, strings.TrimSpace(trimmed[len("save "):])), nil
	case strings.HasPrefix(lower, "list resources"):
		return s.handleListResources(ctx, strings.TrimSpace(trimmed[len("list resources"):])), nil
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
			Message: "Unsupported command. Use: create/list category, resource/save, search, semantic search, create/list todo, create/list reminder, graph.",
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

func (s *ChatService) handleListTodos(ctx context.Context, payload string) ChatResult {
	parts := parsePipePayload(payload)
	limit := parseLimit(parts["limit"], 20)
	items, err := s.todos.List(ctx, limit, 0)
	if err != nil {
		return ChatResult{Action: "todos_error", Message: err.Error()}
	}
	return ChatResult{Action: "todos_list", Message: "Todos loaded", Todos: items}
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

func (s *ChatService) handleListResources(ctx context.Context, payload string) ChatResult {
	parts := parsePipePayload(payload)
	limit := parseLimit(parts["limit"], 20)
	items, err := s.resources.List(ctx, limit, 0)
	if err != nil {
		return ChatResult{Action: "resources_error", Message: err.Error()}
	}
	return ChatResult{Action: "resources_list", Message: "Resources loaded", Resources: items}
}

func (s *ChatService) handleSearchResources(ctx context.Context, payload string) ChatResult {
	parts := parsePipePayload(payload)
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
	parts := parsePipePayload(payload)
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

func (s *ChatService) handleListReminders(ctx context.Context, payload string) ChatResult {
	parts := parsePipePayload(payload)
	limit := parseLimit(parts["limit"], 20)
	items, err := s.reminders.List(ctx, limit, 0)
	if err != nil {
		return ChatResult{Action: "reminders_error", Message: err.Error()}
	}
	return ChatResult{Action: "reminders_list", Message: "Reminders loaded", Reminders: items}
}

func (s *ChatService) handleGraph(ctx context.Context, payload string) ChatResult {
	if s.graph == nil {
		return ChatResult{Action: "graph_error", Message: "graph service is not configured"}
	}

	parts := parsePipePayload(payload)
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
