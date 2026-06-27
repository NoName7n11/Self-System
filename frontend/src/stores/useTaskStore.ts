import { create } from "zustand";

import {
  createReminder,
  createTodo,
  deleteReminder,
  deleteTodo,
  listReminders,
  listTodos,
  normalizeReminder,
  normalizeTodo,
  updateReminder,
  updateTodo,
} from "../api/client";
import { ipcCall } from "../lib/ipc";
import { demoTasksAsTodos } from "../lib/demoData";
import type { ReminderDraft, ReminderItem, ReminderStatus, TodoDraft, TodoItem, TodoStatus } from "../types";

const defaultTodoDraft: TodoDraft = {
  title: "",
  details: "",
  dueAt: "",
  status: "open",
  resourceId: "",
};

const defaultReminderDraft: ReminderDraft = {
  title: "",
  message: "",
  remindAt: "",
  status: "scheduled",
  resourceId: "",
};

function sanitizeTodoStatus(status: string): TodoStatus {
  if (status === "in_progress" || status === "done") {
    return status;
  }
  return "open";
}

function sanitizeReminderStatus(status: string): ReminderStatus {
  if (status === "sent" || status === "dismissed") {
    return status;
  }
  return "scheduled";
}

function toDateTimeLocalValue(timestamp: string): string {
  const parsed = new Date(timestamp);
  if (Number.isNaN(parsed.getTime())) {
    return "";
  }

  const pad = (value: number) => String(value).padStart(2, "0");
  const year = parsed.getFullYear();
  const month = pad(parsed.getMonth() + 1);
  const day = pad(parsed.getDate());
  const hour = pad(parsed.getHours());
  const minute = pad(parsed.getMinutes());

  return `${year}-${month}-${day}T${hour}:${minute}`;
}

function toRFC3339(value: string): string | null {
  const trimmed = value.trim();
  if (trimmed === "") {
    return "";
  }

  const parsed = new Date(trimmed);
  if (Number.isNaN(parsed.getTime())) {
    return null;
  }

  return parsed.toISOString();
}

function todoDraftFromItem(todo: TodoItem): TodoDraft {
  return {
    title: todo.title,
    details: todo.details,
    dueAt: toDateTimeLocalValue(todo.dueAt),
    status: sanitizeTodoStatus(todo.status),
    resourceId: todo.resourceId,
  };
}

function reminderDraftFromItem(reminder: ReminderItem): ReminderDraft {
  return {
    title: reminder.title,
    message: reminder.message,
    remindAt: toDateTimeLocalValue(reminder.remindAt),
    status: sanitizeReminderStatus(reminder.status),
    resourceId: reminder.resourceId,
  };
}

interface TaskState {
  todos: TodoItem[];
  reminders: ReminderItem[];
  isLoadingTodos: boolean;
  isLoadingReminders: boolean;
  error: string | null;
  selectedTodoId: string | null;
  selectedReminderId: string | null;
  todoDraft: TodoDraft;
  reminderDraft: ReminderDraft;
  loadTodos: (options?: { silent?: boolean }) => Promise<void>;
  loadReminders: (options?: { silent?: boolean }) => Promise<void>;
  loadAll: (options?: { silent?: boolean }) => Promise<void>;
  selectTodo: (todoId: string | null) => void;
  selectReminder: (reminderId: string | null) => void;
  updateTodoDraft: (field: keyof TodoDraft, value: string) => void;
  updateReminderDraft: (field: keyof ReminderDraft, value: string) => void;
  resetTodoDraft: () => void;
  resetReminderDraft: () => void;
  toggleTodo: (todoId: string) => void;
  quickAddTask: (cat?: string) => void;
  addTodo: () => Promise<void>;
  updateSelectedTodo: () => Promise<void>;
  deleteSelectedTodo: () => Promise<void>;
  setSelectedTodoStatus: (status: TodoStatus) => Promise<void>;
  addReminder: () => Promise<void>;
  updateSelectedReminder: () => Promise<void>;
  deleteSelectedReminder: () => Promise<void>;
  setSelectedReminderStatus: (status: ReminderStatus) => Promise<void>;
}

export const useTaskStore = create<TaskState>((set, get) => ({
  todos: [],
  reminders: [],
  isLoadingTodos: false,
  isLoadingReminders: false,
  error: null,
  selectedTodoId: null,
  selectedReminderId: null,
  todoDraft: defaultTodoDraft,
  reminderDraft: defaultReminderDraft,

  loadTodos: async (options) => {
    const silent = options?.silent === true;
    if (silent) {
      set({ error: null });
    } else {
      set({ isLoadingTodos: true, error: null });
    }

    try {
      const rawRows = await ipcCall<unknown[]>("desktop.App.GetTodos", [50, 0], () => listTodos());
      const fetched = rawRows.map(normalizeTodo);
      // ponytail: demo seed fallback when backend has no todos
      const rows = fetched.length === 0 ? demoTasksAsTodos() : fetched;
      const selectedTodoId = get().selectedTodoId;

      set((state) => {
        const selected = rows.find((item) => item.id === selectedTodoId) ?? null;
        return {
          todos: rows,
          isLoadingTodos: silent ? state.isLoadingTodos : false,
          error: null,
          selectedTodoId: selected?.id ?? null,
          todoDraft: selected ? todoDraftFromItem(selected) : state.todoDraft,
        };
      });
    } catch {
      // backend unreachable → demo todos so the dock Tasks tab is populated
      set((state) => ({ todos: demoTasksAsTodos(), isLoadingTodos: silent ? state.isLoadingTodos : false, error: null }));
    }
  },

  loadReminders: async (options) => {
    const silent = options?.silent === true;
    if (silent) {
      set({ error: null });
    } else {
      set({ isLoadingReminders: true, error: null });
    }

    try {
      const rawRows = await ipcCall<unknown[]>("desktop.App.GetReminders", [50, 0], () => listReminders());
      const rows = rawRows.map(normalizeReminder);
      const selectedReminderId = get().selectedReminderId;

      set((state) => {
        const selected = rows.find((item) => item.id === selectedReminderId) ?? null;
        return {
          reminders: rows,
          isLoadingReminders: silent ? state.isLoadingReminders : false,
          error: null,
          selectedReminderId: selected?.id ?? null,
          reminderDraft: selected ? reminderDraftFromItem(selected) : state.reminderDraft,
        };
      });
    } catch (error) {
      const message = error instanceof Error ? error.message : "Failed to load reminders";
      set((state) => ({ isLoadingReminders: silent ? state.isLoadingReminders : false, error: message }));
    }
  },

  loadAll: async (options) => {
    await Promise.all([get().loadTodos(options), get().loadReminders(options)]);
  },

  selectTodo: (todoId) => {
    if (!todoId) {
      set({ selectedTodoId: null, todoDraft: defaultTodoDraft });
      return;
    }

    const selected = get().todos.find((item) => item.id === todoId);
    if (!selected) {
      set({ selectedTodoId: null, todoDraft: defaultTodoDraft });
      return;
    }

    set({
      selectedTodoId: selected.id,
      todoDraft: todoDraftFromItem(selected),
      error: null,
    });
  },

  selectReminder: (reminderId) => {
    if (!reminderId) {
      set({ selectedReminderId: null, reminderDraft: defaultReminderDraft });
      return;
    }

    const selected = get().reminders.find((item) => item.id === reminderId);
    if (!selected) {
      set({ selectedReminderId: null, reminderDraft: defaultReminderDraft });
      return;
    }

    set({
      selectedReminderId: selected.id,
      reminderDraft: reminderDraftFromItem(selected),
      error: null,
    });
  },

  updateTodoDraft: (field, value) => {
    set((state) => {
      const nextValue = field === "status" ? sanitizeTodoStatus(value) : value;
      return {
        todoDraft: {
          ...state.todoDraft,
          [field]: nextValue,
        } as TodoDraft,
      };
    });
  },

  updateReminderDraft: (field, value) => {
    set((state) => {
      const nextValue = field === "status" ? sanitizeReminderStatus(value) : value;
      return {
        reminderDraft: {
          ...state.reminderDraft,
          [field]: nextValue,
        } as ReminderDraft,
      };
    });
  },

  resetTodoDraft: () => {
    set({ selectedTodoId: null, todoDraft: defaultTodoDraft, error: null });
  },

  resetReminderDraft: () => {
    set({ selectedReminderId: null, reminderDraft: defaultReminderDraft, error: null });
  },

  // local optimistic toggle (done ↔ open) — works in demo mode; mirrors design toggleTask
  toggleTodo: (todoId) => {
    set((state) => ({
      todos: state.todos.map((t) =>
        t.id === todoId ? { ...t, status: t.status === "done" ? "open" : "done" } : t,
      ),
    }));
  },

  // local quick-add (design "+ NEW" / newTask) — prepends a blank task
  quickAddTask: (cat = "research") => {
    const now = `nt${Date.now()}`;
    set((state) => ({
      todos: [
        { id: now, title: "New task", details: "", status: "open", dueAt: "—", resourceId: "", cat, createdAt: "", updatedAt: "" },
        ...state.todos,
      ],
    }));
  },

  addTodo: async () => {
    const draft = get().todoDraft;
    const title = draft.title.trim();
    if (title === "") {
      set({ error: "Todo title is required." });
      return;
    }

    const dueAt = toRFC3339(draft.dueAt);
    if (dueAt === null) {
      set({ error: "Todo due date must be a valid date/time." });
      return;
    }

    try {
      const created = normalizeTodo(await ipcCall<unknown>(
        "desktop.App.CreateTodo",
        [title, draft.details, dueAt ?? "", draft.resourceId],
        () => createTodo({ title, details: draft.details, dueAt, resourceId: draft.resourceId })
      ));

      set((state) => ({
        todos: [created, ...state.todos.filter((item) => item.id !== created.id)],
        selectedTodoId: created.id,
        todoDraft: todoDraftFromItem(created),
        error: null,
      }));
    } catch (error) {
      const message = error instanceof Error ? error.message : "Failed to create todo";
      set({ error: message });
    }
  },

  updateSelectedTodo: async () => {
    const selectedTodoId = get().selectedTodoId;
    if (!selectedTodoId) {
      set({ error: "Select a todo before updating." });
      return;
    }

    const draft = get().todoDraft;
    const title = draft.title.trim();
    if (title === "") {
      set({ error: "Todo title is required." });
      return;
    }

    const dueAt = toRFC3339(draft.dueAt);
    if (dueAt === null) {
      set({ error: "Todo due date must be a valid date/time." });
      return;
    }

    try {
      const updated = normalizeTodo(await ipcCall<unknown>(
        "desktop.App.UpdateTodo",
        [selectedTodoId, title, draft.details, dueAt ?? "", sanitizeTodoStatus(draft.status), draft.resourceId],
        () => updateTodo(selectedTodoId, {
          title,
          details: draft.details,
          dueAt,
          status: sanitizeTodoStatus(draft.status),
          resourceId: draft.resourceId,
        })
      ));

      set((state) => ({
        todos: state.todos.map((item) => (item.id === updated.id ? updated : item)),
        selectedTodoId: updated.id,
        todoDraft: todoDraftFromItem(updated),
        error: null,
      }));
    } catch (error) {
      const message = error instanceof Error ? error.message : "Failed to update todo";
      set({ error: message });
    }
  },

  deleteSelectedTodo: async () => {
    const selectedTodoId = get().selectedTodoId;
    if (!selectedTodoId) {
      set({ error: "Select a todo before deleting." });
      return;
    }

    try {
      await ipcCall<boolean | void>("desktop.App.DeleteTodo", [selectedTodoId], () => deleteTodo(selectedTodoId));
      set((state) => ({
        todos: state.todos.filter((item) => item.id !== selectedTodoId),
        selectedTodoId: null,
        todoDraft: defaultTodoDraft,
        error: null,
      }));
    } catch (error) {
      const message = error instanceof Error ? error.message : "Failed to delete todo";
      set({ error: message });
    }
  },

  setSelectedTodoStatus: async (status) => {
    const selectedTodoId = get().selectedTodoId;
    if (!selectedTodoId) {
      set({ error: "Select a todo before changing status." });
      return;
    }

    set((state) => ({
      todoDraft: {
        ...state.todoDraft,
        status,
      },
    }));

    await get().updateSelectedTodo();
  },

  addReminder: async () => {
    const draft = get().reminderDraft;
    const title = draft.title.trim();
    if (title === "") {
      set({ error: "Reminder title is required." });
      return;
    }

    const remindAt = toRFC3339(draft.remindAt);
    if (remindAt === null || remindAt === "") {
      set({ error: "Reminder time is required and must be valid." });
      return;
    }

    try {
      const created = normalizeReminder(await ipcCall<unknown>(
        "desktop.App.CreateReminder",
        [title, draft.message, remindAt, draft.resourceId],
        () => createReminder({ title, message: draft.message, remindAt, resourceId: draft.resourceId })
      ));

      set((state) => ({
        reminders: [created, ...state.reminders.filter((item) => item.id !== created.id)],
        selectedReminderId: created.id,
        reminderDraft: reminderDraftFromItem(created),
        error: null,
      }));
    } catch (error) {
      const message = error instanceof Error ? error.message : "Failed to create reminder";
      set({ error: message });
    }
  },

  updateSelectedReminder: async () => {
    const selectedReminderId = get().selectedReminderId;
    if (!selectedReminderId) {
      set({ error: "Select a reminder before updating." });
      return;
    }

    const draft = get().reminderDraft;
    const title = draft.title.trim();
    if (title === "") {
      set({ error: "Reminder title is required." });
      return;
    }

    const remindAt = toRFC3339(draft.remindAt);
    if (remindAt === null || remindAt === "") {
      set({ error: "Reminder time is required and must be valid." });
      return;
    }

    try {
      const updated = normalizeReminder(await ipcCall<unknown>(
        "desktop.App.UpdateReminder",
        [selectedReminderId, title, draft.message, remindAt, sanitizeReminderStatus(draft.status), draft.resourceId],
        () => updateReminder(selectedReminderId, {
          title,
          message: draft.message,
          remindAt,
          status: sanitizeReminderStatus(draft.status),
          resourceId: draft.resourceId,
        })
      ));

      set((state) => ({
        reminders: state.reminders.map((item) => (item.id === updated.id ? updated : item)),
        selectedReminderId: updated.id,
        reminderDraft: reminderDraftFromItem(updated),
        error: null,
      }));
    } catch (error) {
      const message = error instanceof Error ? error.message : "Failed to update reminder";
      set({ error: message });
    }
  },

  deleteSelectedReminder: async () => {
    const selectedReminderId = get().selectedReminderId;
    if (!selectedReminderId) {
      set({ error: "Select a reminder before deleting." });
      return;
    }

    try {
      await ipcCall<boolean | void>("desktop.App.DeleteReminder", [selectedReminderId], () => deleteReminder(selectedReminderId));
      set((state) => ({
        reminders: state.reminders.filter((item) => item.id !== selectedReminderId),
        selectedReminderId: null,
        reminderDraft: defaultReminderDraft,
        error: null,
      }));
    } catch (error) {
      const message = error instanceof Error ? error.message : "Failed to delete reminder";
      set({ error: message });
    }
  },

  setSelectedReminderStatus: async (status) => {
    const selectedReminderId = get().selectedReminderId;
    if (!selectedReminderId) {
      set({ error: "Select a reminder before changing status." });
      return;
    }

    set((state) => ({
      reminderDraft: {
        ...state.reminderDraft,
        status,
      },
    }));

    await get().updateSelectedReminder();
  },
}));
