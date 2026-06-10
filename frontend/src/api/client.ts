import type {
  ChatCommandResult,
  ReminderItem,
  ReminderStatus,
  ResourceItem,
  TodoItem,
  TodoStatus,
} from "../types";

export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "http://127.0.0.1:8080";
export const DEFAULT_SYNC_WS_PATH = "/api/v1/sync/ws";

interface ApiEnvelope<T> {
  data?: T;
  error?: string;
  message?: string;
}

interface ResourcePayload {
  url?: string;
  title?: string;
  summary?: string;
  categoryName?: string;
}

interface TodoPayload {
  title?: string;
  details?: string;
  dueAt?: string;
  status?: TodoStatus;
  resourceId?: string;
}

interface ReminderPayload {
  title?: string;
  message?: string;
  remindAt?: string;
  status?: ReminderStatus;
  resourceId?: string;
}

export function resolveSyncWebSocketURL(): string {
  const configuredURL = (import.meta.env.VITE_SYNC_WS_URL ?? "").trim();
  if (configuredURL !== "") {
    return configuredURL;
  }

  const configuredPath = (import.meta.env.VITE_SYNC_WS_PATH ?? DEFAULT_SYNC_WS_PATH).trim();
  const normalizedPath = configuredPath.startsWith("/") ? configuredPath : `/${configuredPath}`;

  const normalizedBaseURL = API_BASE_URL.trim().replace(/\/+$/, "");
  const resolved = new URL(`${normalizedBaseURL}${normalizedPath}`);

  if (resolved.protocol === "https:") {
    resolved.protocol = "wss:";
  } else if (resolved.protocol === "http:") {
    resolved.protocol = "ws:";
  }

  return resolved.toString();
}

function firstString(...values: unknown[]): string {
  for (const value of values) {
    if (typeof value === "string" && value.trim() !== "") {
      return value.trim();
    }
  }
  return "";
}

function toBool(value: unknown): boolean {
  if (typeof value === "boolean") {
    return value;
  }
  if (typeof value === "number") {
    return value !== 0;
  }
  if (typeof value === "string") {
    const normalized = value.trim().toLowerCase();
    return normalized === "true" || normalized === "1" || normalized === "yes";
  }
  return false;
}

export function normalizeResource(raw: unknown): ResourceItem {
  const resource = (raw ?? {}) as Record<string, unknown>;

  return {
    id: firstString(resource.id, resource.ID),
    url: firstString(resource.url, resource.URL),
    host: firstString(resource.host, resource.Host),
    title: firstString(resource.title, resource.Title),
    summary: firstString(resource.summary, resource.Summary),
    categoryId: firstString(resource.category_id, resource.categoryId, resource.CategoryID),
    categoryName: firstString(resource.category_name, resource.categoryName, resource.CategoryName),
    userOverride: toBool(resource.user_override ?? resource.userOverride ?? resource.UserOverride),
    createdAt: firstString(resource.created_at, resource.createdAt, resource.CreatedAt),
    updatedAt: firstString(resource.updated_at, resource.updatedAt, resource.UpdatedAt),
  };
}

export function normalizeTodo(raw: unknown): TodoItem {
  const todo = (raw ?? {}) as Record<string, unknown>;

  return {
    id: firstString(todo.id, todo.ID),
    title: firstString(todo.title, todo.Title),
    details: firstString(todo.details, todo.Details),
    status: firstString(todo.status, todo.Status) as TodoStatus,
    dueAt: firstString(todo.due_at, todo.dueAt, todo.DueAt),
    resourceId: firstString(todo.resource_id, todo.resourceId, todo.ResourceID),
    createdAt: firstString(todo.created_at, todo.createdAt, todo.CreatedAt),
    updatedAt: firstString(todo.updated_at, todo.updatedAt, todo.UpdatedAt),
  };
}

export function normalizeReminder(raw: unknown): ReminderItem {
  const reminder = (raw ?? {}) as Record<string, unknown>;

  return {
    id: firstString(reminder.id, reminder.ID),
    title: firstString(reminder.title, reminder.Title),
    message: firstString(reminder.message, reminder.Message),
    remindAt: firstString(reminder.remind_at, reminder.remindAt, reminder.RemindAt),
    status: firstString(reminder.status, reminder.Status) as ReminderStatus,
    resourceId: firstString(reminder.resource_id, reminder.resourceId, reminder.ResourceID),
    createdAt: firstString(reminder.created_at, reminder.createdAt, reminder.CreatedAt),
    updatedAt: firstString(reminder.updated_at, reminder.updatedAt, reminder.UpdatedAt),
  };
}

async function apiRequest<T>(path: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...(options?.headers ?? {}),
    },
  });

  const bodyText = await response.text();
  let parsed: ApiEnvelope<T> | null = null;

  if (bodyText.trim() !== "") {
    try {
      parsed = JSON.parse(bodyText) as ApiEnvelope<T>;
    } catch {
      parsed = null;
    }
  }

  if (!response.ok) {
    const message = parsed?.error?.trim() || parsed?.message?.trim() || `Request failed (${response.status})`;
    throw new Error(message);
  }

  if (!parsed || parsed.data === undefined) {
    throw new Error("API response did not include data");
  }

  return parsed.data;
}

async function apiRequestNoData(path: string, options?: RequestInit): Promise<void> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...(options?.headers ?? {}),
    },
  });

  const bodyText = await response.text();
  let parsed: ApiEnvelope<unknown> | null = null;

  if (bodyText.trim() !== "") {
    try {
      parsed = JSON.parse(bodyText) as ApiEnvelope<unknown>;
    } catch {
      parsed = null;
    }
  }

  if (!response.ok) {
    const message = parsed?.error?.trim() || parsed?.message?.trim() || `Request failed (${response.status})`;
    throw new Error(message);
  }
}

export async function listResources(limit = 200): Promise<ResourceItem[]> {
  const rows = await apiRequest<unknown[]>(`/api/v1/resources?limit=${encodeURIComponent(String(limit))}`);
  if (!Array.isArray(rows)) {
    return [];
  }

  return rows
    .map((row) => normalizeResource(row))
    .filter((resource) => resource.id !== "");
}

export async function createResource(payload: ResourcePayload): Promise<ResourceItem> {
  const body: Record<string, string> = {};

  if (payload.url?.trim()) {
    body.url = payload.url.trim();
  }
  if (payload.title?.trim()) {
    body.title = payload.title.trim();
  }
  if (payload.summary?.trim()) {
    body.summary = payload.summary.trim();
  }
  if (payload.categoryName?.trim()) {
    body.category_name = payload.categoryName.trim();
  }

  const created = await apiRequest<unknown>("/api/v1/resources", {
    method: "POST",
    body: JSON.stringify(body),
  });

  return normalizeResource(created);
}

export async function updateResource(resourceId: string, payload: ResourcePayload): Promise<ResourceItem> {
  const body: Record<string, string> = {};

  if (payload.url?.trim()) {
    body.url = payload.url.trim();
  }
  if (payload.title?.trim()) {
    body.title = payload.title.trim();
  }
  if (payload.summary?.trim()) {
    body.summary = payload.summary.trim();
  }
  if (payload.categoryName?.trim()) {
    body.category_name = payload.categoryName.trim();
  }

  const updated = await apiRequest<unknown>(`/api/v1/resources/${encodeURIComponent(resourceId)}`, {
    method: "PUT",
    body: JSON.stringify(body),
  });

  return normalizeResource(updated);
}

export async function deleteResource(resourceId: string): Promise<void> {
  await apiRequestNoData(`/api/v1/resources/${encodeURIComponent(resourceId)}`, {
    method: "DELETE",
  });
}

export async function listTodos(limit = 200): Promise<TodoItem[]> {
  const rows = await apiRequest<unknown[]>(`/api/v1/todos?limit=${encodeURIComponent(String(limit))}`);
  if (!Array.isArray(rows)) {
    return [];
  }

  return rows
    .map((row) => normalizeTodo(row))
    .filter((todo) => todo.id !== "");
}

export async function createTodo(payload: TodoPayload): Promise<TodoItem> {
  const body: Record<string, string> = {};

  if (payload.title?.trim()) {
    body.title = payload.title.trim();
  }
  if (payload.details?.trim()) {
    body.details = payload.details.trim();
  }
  if (payload.dueAt !== undefined) {
    body.due_at = payload.dueAt.trim();
  }
  if (payload.resourceId !== undefined) {
    body.resource_id = payload.resourceId.trim();
  }

  const created = await apiRequest<unknown>("/api/v1/todos", {
    method: "POST",
    body: JSON.stringify(body),
  });

  return normalizeTodo(created);
}

export async function updateTodo(todoId: string, payload: TodoPayload): Promise<TodoItem> {
  const body: Record<string, string> = {};

  if (payload.title !== undefined) {
    body.title = payload.title.trim();
  }
  if (payload.details !== undefined) {
    body.details = payload.details.trim();
  }
  if (payload.status !== undefined) {
    body.status = payload.status;
  }
  if (payload.dueAt !== undefined) {
    body.due_at = payload.dueAt.trim();
  }
  if (payload.resourceId !== undefined) {
    body.resource_id = payload.resourceId.trim();
  }

  const updated = await apiRequest<unknown>(`/api/v1/todos/${encodeURIComponent(todoId)}`, {
    method: "PUT",
    body: JSON.stringify(body),
  });

  return normalizeTodo(updated);
}

export async function deleteTodo(todoId: string): Promise<void> {
  await apiRequestNoData(`/api/v1/todos/${encodeURIComponent(todoId)}`, {
    method: "DELETE",
  });
}

export async function listReminders(limit = 200): Promise<ReminderItem[]> {
  const rows = await apiRequest<unknown[]>(`/api/v1/reminders?limit=${encodeURIComponent(String(limit))}`);
  if (!Array.isArray(rows)) {
    return [];
  }

  return rows
    .map((row) => normalizeReminder(row))
    .filter((reminder) => reminder.id !== "");
}

export async function createReminder(payload: ReminderPayload): Promise<ReminderItem> {
  const body: Record<string, string> = {};

  if (payload.title?.trim()) {
    body.title = payload.title.trim();
  }
  if (payload.message?.trim()) {
    body.message = payload.message.trim();
  }
  if (payload.remindAt !== undefined) {
    body.remind_at = payload.remindAt.trim();
  }
  if (payload.resourceId !== undefined) {
    body.resource_id = payload.resourceId.trim();
  }

  const created = await apiRequest<unknown>("/api/v1/reminders", {
    method: "POST",
    body: JSON.stringify(body),
  });

  return normalizeReminder(created);
}

export async function updateReminder(reminderId: string, payload: ReminderPayload): Promise<ReminderItem> {
  const body: Record<string, string> = {};

  if (payload.title !== undefined) {
    body.title = payload.title.trim();
  }
  if (payload.message !== undefined) {
    body.message = payload.message.trim();
  }
  if (payload.remindAt !== undefined) {
    body.remind_at = payload.remindAt.trim();
  }
  if (payload.status !== undefined) {
    body.status = payload.status;
  }
  if (payload.resourceId !== undefined) {
    body.resource_id = payload.resourceId.trim();
  }

  const updated = await apiRequest<unknown>(`/api/v1/reminders/${encodeURIComponent(reminderId)}`, {
    method: "PUT",
    body: JSON.stringify(body),
  });

  return normalizeReminder(updated);
}

export async function deleteReminder(reminderId: string): Promise<void> {
  await apiRequestNoData(`/api/v1/reminders/${encodeURIComponent(reminderId)}`, {
    method: "DELETE",
  });
}

export async function sendChatCommand(message: string): Promise<ChatCommandResult> {
  const result = await apiRequest<unknown>("/api/v1/chat/commands", {
    method: "POST",
    body: JSON.stringify({ message }),
  });

  const payload = (result ?? {}) as Record<string, unknown>;
  return {
    action: firstString(payload.action, payload.Action),
    message: firstString(payload.message, payload.Message),
  };
}
