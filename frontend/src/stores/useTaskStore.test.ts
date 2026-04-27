import { beforeEach, describe, expect, it, vi } from "vitest";

import { useTaskStore } from "./useTaskStore";
import {
  createReminder,
  createTodo,
  deleteReminder,
  deleteTodo,
  listReminders,
  listTodos,
  updateReminder,
  updateTodo,
} from "../api/client";

vi.mock("../api/client", () => ({
  listTodos: vi.fn(),
  listReminders: vi.fn(),
  createTodo: vi.fn(),
  updateTodo: vi.fn(),
  deleteTodo: vi.fn(),
  createReminder: vi.fn(),
  updateReminder: vi.fn(),
  deleteReminder: vi.fn(),
}));

function resetTaskStore() {
  useTaskStore.setState({
    todos: [],
    reminders: [],
    isLoadingTodos: false,
    isLoadingReminders: false,
    error: null,
    selectedTodoId: null,
    selectedReminderId: null,
    todoDraft: {
      title: "",
      details: "",
      dueAt: "",
      status: "open",
      resourceId: "",
    },
    reminderDraft: {
      title: "",
      message: "",
      remindAt: "",
      status: "scheduled",
      resourceId: "",
    },
  });
}

function seedSelectedTodo() {
  useTaskStore.setState({
    todos: [
      {
        id: "todo-1",
        title: "Draft roadmap",
        details: "link task to resource",
        status: "open",
        dueAt: "2026-04-20T10:30:00.000Z",
        resourceId: "res-1",
        createdAt: "2026-04-18T12:00:00.000Z",
        updatedAt: "2026-04-18T12:00:00.000Z",
      },
    ],
    selectedTodoId: "todo-1",
    todoDraft: {
      title: "Draft roadmap",
      details: "link task to resource",
      dueAt: "2026-04-20T10:30",
      status: "open",
      resourceId: "res-1",
    },
    error: null,
  });
}

function seedSelectedReminder() {
  useTaskStore.setState({
    reminders: [
      {
        id: "rem-1",
        title: "Check task links",
        message: "verify related resource",
        remindAt: "2026-04-21T09:00:00.000Z",
        status: "scheduled",
        resourceId: "res-1",
        createdAt: "2026-04-18T12:00:00.000Z",
        updatedAt: "2026-04-18T12:00:00.000Z",
      },
    ],
    selectedReminderId: "rem-1",
    reminderDraft: {
      title: "Check task links",
      message: "verify related resource",
      remindAt: "2026-04-21T09:00",
      status: "scheduled",
      resourceId: "res-1",
    },
    error: null,
  });
}

describe("useTaskStore", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetTaskStore();
    vi.mocked(listTodos).mockResolvedValue([]);
    vi.mocked(listReminders).mockResolvedValue([]);
    vi.mocked(createTodo).mockResolvedValue({
      id: "todo-1",
      title: "Draft roadmap",
      details: "link task to resource",
      status: "open",
      dueAt: "2026-04-20T10:30:00.000Z",
      resourceId: "res-1",
      createdAt: "2026-04-18T12:00:00.000Z",
      updatedAt: "2026-04-18T12:00:00.000Z",
    });
    vi.mocked(updateTodo).mockResolvedValue({
      id: "todo-1",
      title: "Draft roadmap",
      details: "updated",
      status: "in_progress",
      dueAt: "2026-04-20T11:30:00.000Z",
      resourceId: "res-2",
      createdAt: "2026-04-18T12:00:00.000Z",
      updatedAt: "2026-04-18T12:30:00.000Z",
    });
    vi.mocked(deleteTodo).mockResolvedValue(undefined);
    vi.mocked(createReminder).mockResolvedValue({
      id: "rem-1",
      title: "Check task links",
      message: "verify related resource",
      remindAt: "2026-04-21T09:00:00.000Z",
      status: "scheduled",
      resourceId: "res-1",
      createdAt: "2026-04-18T12:00:00.000Z",
      updatedAt: "2026-04-18T12:00:00.000Z",
    });
    vi.mocked(updateReminder).mockResolvedValue({
      id: "rem-1",
      title: "Check task links",
      message: "sent",
      remindAt: "2026-04-21T10:00:00.000Z",
      status: "sent",
      resourceId: "res-2",
      createdAt: "2026-04-18T12:00:00.000Z",
      updatedAt: "2026-04-18T12:30:00.000Z",
    });
    vi.mocked(deleteReminder).mockResolvedValue(undefined);
  });

  it("creates todo with linked resource id", async () => {
    useTaskStore.getState().updateTodoDraft("title", "Draft roadmap");
    useTaskStore.getState().updateTodoDraft("details", "link task to resource");
    useTaskStore.getState().updateTodoDraft("dueAt", "2026-04-20T10:30");
    useTaskStore.getState().updateTodoDraft("resourceId", "res-1");

    await useTaskStore.getState().addTodo();

    expect(createTodo).toHaveBeenCalledTimes(1);
    expect(createTodo).toHaveBeenCalledWith(
      expect.objectContaining({
        title: "Draft roadmap",
        details: "link task to resource",
        resourceId: "res-1",
      }),
    );

    const state = useTaskStore.getState();
    expect(state.todos).toHaveLength(1);
    expect(state.selectedTodoId).toBe("todo-1");
  });

  it("creates reminder with linked resource id", async () => {
    useTaskStore.getState().updateReminderDraft("title", "Check task links");
    useTaskStore.getState().updateReminderDraft("message", "verify related resource");
    useTaskStore.getState().updateReminderDraft("remindAt", "2026-04-21T09:00");
    useTaskStore.getState().updateReminderDraft("resourceId", "res-1");

    await useTaskStore.getState().addReminder();

    expect(createReminder).toHaveBeenCalledTimes(1);
    expect(createReminder).toHaveBeenCalledWith(
      expect.objectContaining({
        title: "Check task links",
        message: "verify related resource",
        resourceId: "res-1",
      }),
    );

    const state = useTaskStore.getState();
    expect(state.reminders).toHaveLength(1);
    expect(state.selectedReminderId).toBe("rem-1");
  });

  it("sets todo create error and keeps list unchanged", async () => {
    vi.mocked(createTodo).mockRejectedValueOnce(new Error("mock create todo error"));

    useTaskStore.getState().updateTodoDraft("title", "Create todo fails");
    useTaskStore.getState().updateTodoDraft("details", "failure path");
    useTaskStore.getState().updateTodoDraft("dueAt", "2026-04-20T10:30");
    useTaskStore.getState().updateTodoDraft("resourceId", "res-1");

    await useTaskStore.getState().addTodo();

    const state = useTaskStore.getState();
    expect(state.error).toBe("mock create todo error");
    expect(state.todos).toHaveLength(0);
    expect(state.selectedTodoId).toBeNull();
  });

  it("sets todo update error and preserves existing todo", async () => {
    seedSelectedTodo();
    vi.mocked(updateTodo).mockRejectedValueOnce(new Error("mock update todo error"));

    useTaskStore.getState().updateTodoDraft("title", "Draft roadmap updated");
    useTaskStore.getState().updateTodoDraft("status", "in_progress");

    await useTaskStore.getState().updateSelectedTodo();

    expect(updateTodo).toHaveBeenCalledWith(
      "todo-1",
      expect.objectContaining({
        title: "Draft roadmap updated",
        status: "in_progress",
      }),
    );

    const state = useTaskStore.getState();
    expect(state.error).toBe("mock update todo error");
    expect(state.todos).toHaveLength(1);
    expect(state.todos[0]?.title).toBe("Draft roadmap");
    expect(state.selectedTodoId).toBe("todo-1");
  });

  it("sets todo delete error and keeps selected todo", async () => {
    seedSelectedTodo();
    vi.mocked(deleteTodo).mockRejectedValueOnce(new Error("mock delete todo error"));

    await useTaskStore.getState().deleteSelectedTodo();

    expect(deleteTodo).toHaveBeenCalledWith("todo-1");

    const state = useTaskStore.getState();
    expect(state.error).toBe("mock delete todo error");
    expect(state.todos).toHaveLength(1);
    expect(state.selectedTodoId).toBe("todo-1");
  });

  it("sets reminder create error and keeps list unchanged", async () => {
    vi.mocked(createReminder).mockRejectedValueOnce(new Error("mock create reminder error"));

    useTaskStore.getState().updateReminderDraft("title", "Create reminder fails");
    useTaskStore.getState().updateReminderDraft("message", "failure path");
    useTaskStore.getState().updateReminderDraft("remindAt", "2026-04-21T09:00");
    useTaskStore.getState().updateReminderDraft("resourceId", "res-1");

    await useTaskStore.getState().addReminder();

    const state = useTaskStore.getState();
    expect(state.error).toBe("mock create reminder error");
    expect(state.reminders).toHaveLength(0);
    expect(state.selectedReminderId).toBeNull();
  });

  it("sets reminder update error and preserves existing reminder", async () => {
    seedSelectedReminder();
    vi.mocked(updateReminder).mockRejectedValueOnce(new Error("mock update reminder error"));

    useTaskStore.getState().updateReminderDraft("message", "updated message");
    useTaskStore.getState().updateReminderDraft("status", "sent");

    await useTaskStore.getState().updateSelectedReminder();

    expect(updateReminder).toHaveBeenCalledWith(
      "rem-1",
      expect.objectContaining({
        message: "updated message",
        status: "sent",
      }),
    );

    const state = useTaskStore.getState();
    expect(state.error).toBe("mock update reminder error");
    expect(state.reminders).toHaveLength(1);
    expect(state.reminders[0]?.message).toBe("verify related resource");
    expect(state.selectedReminderId).toBe("rem-1");
  });

  it("sets reminder delete error and keeps selected reminder", async () => {
    seedSelectedReminder();
    vi.mocked(deleteReminder).mockRejectedValueOnce(new Error("mock delete reminder error"));

    await useTaskStore.getState().deleteSelectedReminder();

    expect(deleteReminder).toHaveBeenCalledWith("rem-1");

    const state = useTaskStore.getState();
    expect(state.error).toBe("mock delete reminder error");
    expect(state.reminders).toHaveLength(1);
    expect(state.selectedReminderId).toBe("rem-1");
  });

  it("sets error when todo list fails", async () => {
    vi.mocked(listTodos).mockRejectedValueOnce(new Error("mock todo list failure"));

    // call loadTodos if available; store should surface the error
    // some store implementations return a promise from loadTodos
    // calling it defensively here
    // @ts-ignore
    await useTaskStore.getState().loadTodos?.();

    const state = useTaskStore.getState();
    expect(state.error).toBe("mock todo list failure");
    expect(state.todos).toHaveLength(0);
  });
});
