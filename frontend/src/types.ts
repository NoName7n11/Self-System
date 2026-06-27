export type ViewMode = "2d" | "3d";

export type NavSection = "graph" | "search" | "chat" | "tasks" | "settings";

export type SyncStatus = "idle" | "connecting" | "connected" | "reconnecting" | "offline";

export type TodoStatus = "open" | "in_progress" | "done";

export type ReminderStatus = "scheduled" | "sent" | "dismissed";

export type ResourceType = "pdf" | "link" | "note" | "doc" | "image";

export type DockTab = "categories" | "chat" | "tasks" | "library" | "archive";

export type GraphView = "graph" | "map" | "progress";

export type LeftView = "home" | "chat" | "tasks" | "library";

export interface ResourceItem {
  id: string;
  url: string;
  host: string;
  title: string;
  summary: string;
  categoryId: string;
  categoryName: string;
  userOverride: boolean;
  type?: ResourceType;
  createdAt: string;
  updatedAt: string;
}

export interface ResourceDraft {
  url: string;
  title: string;
  summary: string;
  categoryName: string;
}

export interface ResourceFilters {
  query: string;
  category: string;
  viewMode: ViewMode;
  showOverridesOnly: boolean;
}

export interface ChatMessage {
  id: string;
  role: "user" | "assistant";
  content: string;
  createdAt: string;
}

export interface ChatCommandResult {
  action: string;
  message?: string;
}

export interface SyncEvent {
  type: string;
  sequence?: number;
  payload?: Record<string, unknown>;
  timestamp?: string;
}

export interface TodoItem {
  id: string;
  title: string;
  details: string;
  status: TodoStatus;
  dueAt: string;
  resourceId: string;
  cat?: string;
  createdAt: string;
  updatedAt: string;
}

export interface ReminderItem {
  id: string;
  title: string;
  message: string;
  remindAt: string;
  status: ReminderStatus;
  resourceId: string;
  createdAt: string;
  updatedAt: string;
}

export interface TodoDraft {
  title: string;
  details: string;
  dueAt: string;
  status: TodoStatus;
  resourceId: string;
}

export interface ReminderDraft {
  title: string;
  message: string;
  remindAt: string;
  status: ReminderStatus;
  resourceId: string;
}
