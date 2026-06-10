import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

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
  normalizeTodo: vi.fn((raw: unknown) => raw),
  normalizeReminder: vi.fn((raw: unknown) => raw),
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

  it("sanitizes todo status when updating the draft", () => {
    useTaskStore.getState().updateTodoDraft("status", "invalid-status");

    expect(useTaskStore.getState().todoDraft.status).toBe("open");
  });

  it("sanitizes reminder status when updating the draft", () => {
    useTaskStore.getState().updateReminderDraft("status", "invalid-status");

    expect(useTaskStore.getState().reminderDraft.status).toBe("scheduled");
  });

  it("requires a todo title before creating", async () => {
    useTaskStore.getState().updateTodoDraft("dueAt", "2026-04-20T10:30");

    await useTaskStore.getState().addTodo();

    expect(createTodo).not.toHaveBeenCalled();
    expect(useTaskStore.getState().error).toBe("Todo title is required.");
  });

  it("requires a valid todo due date before creating", async () => {
    useTaskStore.getState().updateTodoDraft("title", "Draft roadmap");
    useTaskStore.getState().updateTodoDraft("dueAt", "not-a-date");

    await useTaskStore.getState().addTodo();

    expect(createTodo).not.toHaveBeenCalled();
    expect(useTaskStore.getState().error).toBe("Todo due date must be a valid date/time.");
  });

  it("requires a reminder title before creating", async () => {
    useTaskStore.getState().updateReminderDraft("remindAt", "2026-04-21T09:00");

    await useTaskStore.getState().addReminder();

    expect(createReminder).not.toHaveBeenCalled();
    expect(useTaskStore.getState().error).toBe("Reminder title is required.");
  });

  it("requires a valid reminder time before creating", async () => {
    useTaskStore.getState().updateReminderDraft("title", "Check task links");
    useTaskStore.getState().updateReminderDraft("remindAt", "not-a-date");

    await useTaskStore.getState().addReminder();

    expect(createReminder).not.toHaveBeenCalled();
    expect(useTaskStore.getState().error).toBe("Reminder time is required and must be valid.");
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

  it("sets loading state while todos refresh is pending", async () => {
    let resolveList!: (value: Awaited<ReturnType<typeof listTodos>>) => void;
    const listPromise = new Promise<Awaited<ReturnType<typeof listTodos>>>((resolve) => {
      resolveList = resolve;
    });

    useTaskStore.setState({ isLoadingTodos: false, error: "stale error" });
    vi.mocked(listTodos).mockReturnValueOnce(listPromise);

    const loadPromise = useTaskStore.getState().loadTodos();

    const midState = useTaskStore.getState();
    expect(midState.isLoadingTodos).toBe(true);
    expect(midState.error).toBeNull();

    resolveList([]);
    await loadPromise;

    const finalState = useTaskStore.getState();
    expect(finalState.isLoadingTodos).toBe(false);
    expect(finalState.error).toBeNull();
  });

  it("keeps loading state while silent todos refresh is pending", async () => {
    let resolveList!: (value: Awaited<ReturnType<typeof listTodos>>) => void;
    const listPromise = new Promise<Awaited<ReturnType<typeof listTodos>>>((resolve) => {
      resolveList = resolve;
    });

    useTaskStore.setState({ isLoadingTodos: true, error: "stale error" });
    vi.mocked(listTodos).mockReturnValueOnce(listPromise);

    const loadPromise = useTaskStore.getState().loadTodos({ silent: true });

    const midState = useTaskStore.getState();
    expect(midState.isLoadingTodos).toBe(true);
    expect(midState.error).toBeNull();

    resolveList([]);
    await loadPromise;

    const finalState = useTaskStore.getState();
    expect(finalState.isLoadingTodos).toBe(true);
    expect(finalState.error).toBeNull();
  });

  it("sets loading state while reminders refresh is pending", async () => {
    let resolveList!: (value: Awaited<ReturnType<typeof listReminders>>) => void;
    const listPromise = new Promise<Awaited<ReturnType<typeof listReminders>>>((resolve) => {
      resolveList = resolve;
    });

    useTaskStore.setState({ isLoadingReminders: false, error: "stale error" });
    vi.mocked(listReminders).mockReturnValueOnce(listPromise);

    const loadPromise = useTaskStore.getState().loadReminders();

    const midState = useTaskStore.getState();
    expect(midState.isLoadingReminders).toBe(true);
    expect(midState.error).toBeNull();

    resolveList([]);
    await loadPromise;

    const finalState = useTaskStore.getState();
    expect(finalState.isLoadingReminders).toBe(false);
    expect(finalState.error).toBeNull();
  });

  it("keeps loading state while silent reminders refresh is pending", async () => {
    let resolveList!: (value: Awaited<ReturnType<typeof listReminders>>) => void;
    const listPromise = new Promise<Awaited<ReturnType<typeof listReminders>>>((resolve) => {
      resolveList = resolve;
    });

    useTaskStore.setState({ isLoadingReminders: true, error: "stale error" });
    vi.mocked(listReminders).mockReturnValueOnce(listPromise);

    const loadPromise = useTaskStore.getState().loadReminders({ silent: true });

    const midState = useTaskStore.getState();
    expect(midState.isLoadingReminders).toBe(true);
    expect(midState.error).toBeNull();

    resolveList([]);
    await loadPromise;

    const finalState = useTaskStore.getState();
    expect(finalState.isLoadingReminders).toBe(true);
    expect(finalState.error).toBeNull();
  });

  it("updates selected todo status and persists the change", async () => {
    seedSelectedTodo();

    await useTaskStore.getState().setSelectedTodoStatus("done");

    expect(updateTodo).toHaveBeenCalledWith(
      "todo-1",
      expect.objectContaining({
        status: "done",
      }),
    );

    const state = useTaskStore.getState();
    expect(state.selectedTodoId).toBe("todo-1");
    expect(state.todoDraft.status).toBe("in_progress");
  });

  it("sets error when changing todo status without a selection", async () => {
    await useTaskStore.getState().setSelectedTodoStatus("done");

    expect(updateTodo).not.toHaveBeenCalled();
    expect(useTaskStore.getState().error).toBe("Select a todo before changing status.");
  });

  it("updates selected reminder status and persists the change", async () => {
    seedSelectedReminder();

    await useTaskStore.getState().setSelectedReminderStatus("dismissed");

    expect(updateReminder).toHaveBeenCalledWith(
      "rem-1",
      expect.objectContaining({
        status: "dismissed",
      }),
    );

    const state = useTaskStore.getState();
    expect(state.selectedReminderId).toBe("rem-1");
    expect(state.reminderDraft.status).toBe("sent");
  });

  it("sets error when changing reminder status without a selection", async () => {
    await useTaskStore.getState().setSelectedReminderStatus("dismissed");

    expect(updateReminder).not.toHaveBeenCalled();
    expect(useTaskStore.getState().error).toBe("Select a reminder before changing status.");
  });

  it("selects a todo and loads the draft from it", () => {
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
      selectedTodoId: null,
      todoDraft: {
        title: "",
        details: "",
        dueAt: "",
        status: "open",
        resourceId: "",
      },
      error: "stale error",
    });

    useTaskStore.getState().selectTodo("todo-1");

    const state = useTaskStore.getState();
    expect(state.selectedTodoId).toBe("todo-1");
    expect(state.todoDraft.title).toBe("Draft roadmap");
    expect(state.todoDraft.details).toBe("link task to resource");
    expect(state.todoDraft.status).toBe("open");
    expect(state.todoDraft.resourceId).toBe("res-1");
    expect(state.todoDraft.dueAt).not.toBe("");
    expect(state.error).toBeNull();
  });

  it("selects a reminder and loads the draft from it", () => {
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
      selectedReminderId: null,
      reminderDraft: {
        title: "",
        message: "",
        remindAt: "",
        status: "scheduled",
        resourceId: "",
      },
      error: "stale error",
    });

    useTaskStore.getState().selectReminder("rem-1");

    const state = useTaskStore.getState();
    expect(state.selectedReminderId).toBe("rem-1");
    expect(state.reminderDraft.title).toBe("Check task links");
    expect(state.reminderDraft.message).toBe("verify related resource");
    expect(state.reminderDraft.status).toBe("scheduled");
    expect(state.reminderDraft.resourceId).toBe("res-1");
    expect(state.reminderDraft.remindAt).not.toBe("");
    expect(state.error).toBeNull();
  });

  it("resets the todo draft and clears selection", () => {
    seedSelectedTodo();
    useTaskStore.setState({ error: "stale error" });

    useTaskStore.getState().resetTodoDraft();

    const state = useTaskStore.getState();
    expect(state.selectedTodoId).toBeNull();
    expect(state.todoDraft).toEqual({
      title: "",
      details: "",
      dueAt: "",
      status: "open",
      resourceId: "",
    });
    expect(state.error).toBeNull();
  });

  it("resets the reminder draft and clears selection", () => {
    seedSelectedReminder();
    useTaskStore.setState({ error: "stale error" });

    useTaskStore.getState().resetReminderDraft();

    const state = useTaskStore.getState();
    expect(state.selectedReminderId).toBeNull();
    expect(state.reminderDraft).toEqual({
      title: "",
      message: "",
      remindAt: "",
      status: "scheduled",
      resourceId: "",
    });
    expect(state.error).toBeNull();
  });

  it("deletes the selected todo and resets the draft", async () => {
    seedSelectedTodo();

    await useTaskStore.getState().deleteSelectedTodo();

    expect(deleteTodo).toHaveBeenCalledWith("todo-1");

    const state = useTaskStore.getState();
    expect(state.todos).toHaveLength(0);
    expect(state.selectedTodoId).toBeNull();
    expect(state.todoDraft).toEqual({
      title: "",
      details: "",
      dueAt: "",
      status: "open",
      resourceId: "",
    });
    expect(state.error).toBeNull();
  });

  it("deletes the selected reminder and resets the draft", async () => {
    seedSelectedReminder();

    await useTaskStore.getState().deleteSelectedReminder();

    expect(deleteReminder).toHaveBeenCalledWith("rem-1");

    const state = useTaskStore.getState();
    expect(state.reminders).toHaveLength(0);
    expect(state.selectedReminderId).toBeNull();
    expect(state.reminderDraft).toEqual({
      title: "",
      message: "",
      remindAt: "",
      status: "scheduled",
      resourceId: "",
    });
    expect(state.error).toBeNull();
  });

  it("retains the selected todo and refreshes the draft on list success", async () => {
    seedSelectedTodo();
    vi.mocked(listTodos).mockResolvedValueOnce([
      {
        id: "todo-1",
        title: "Draft roadmap refreshed",
        details: "updated details",
        status: "done",
        dueAt: "2026-04-20T12:30:00.000Z",
        resourceId: "res-2",
        createdAt: "2026-04-18T12:00:00.000Z",
        updatedAt: "2026-04-19T08:00:00.000Z",
      },
    ]);

    await useTaskStore.getState().loadTodos({ silent: true });

    const state = useTaskStore.getState();
    expect(state.todos).toHaveLength(1);
    expect(state.selectedTodoId).toBe("todo-1");
    expect(state.todoDraft.title).toBe("Draft roadmap refreshed");
    expect(state.todoDraft.details).toBe("updated details");
    expect(state.todoDraft.status).toBe("done");
    expect(state.todoDraft.resourceId).toBe("res-2");
    expect(state.todoDraft.dueAt).not.toBe("");
    expect(state.error).toBeNull();
  });

  it("clears missing selected todo id but preserves the draft on list success", async () => {
    seedSelectedTodo();
    vi.mocked(listTodos).mockResolvedValueOnce([
      {
        id: "todo-2",
        title: "Replacement todo",
        details: "new list payload",
        status: "open",
        dueAt: "2026-04-22T09:00:00.000Z",
        resourceId: "res-2",
        createdAt: "2026-04-18T12:00:00.000Z",
        updatedAt: "2026-04-19T08:00:00.000Z",
      },
    ]);

    await useTaskStore.getState().loadTodos({ silent: true });

    const state = useTaskStore.getState();
    expect(state.todos).toHaveLength(1);
    expect(state.todos[0]?.id).toBe("todo-2");
    expect(state.selectedTodoId).toBeNull();
    expect(state.todoDraft.title).toBe("Draft roadmap");
    expect(state.todoDraft.dueAt).toBe("2026-04-20T10:30");
  });

  it("retains the selected reminder and refreshes the draft on list success", async () => {
    seedSelectedReminder();
    vi.mocked(listReminders).mockResolvedValueOnce([
      {
        id: "rem-1",
        title: "Check task links refreshed",
        message: "updated reminder",
        remindAt: "2026-04-21T10:15:00.000Z",
        status: "sent",
        resourceId: "res-2",
        createdAt: "2026-04-18T12:00:00.000Z",
        updatedAt: "2026-04-19T08:00:00.000Z",
      },
    ]);

    await useTaskStore.getState().loadReminders({ silent: true });

    const state = useTaskStore.getState();
    expect(state.reminders).toHaveLength(1);
    expect(state.selectedReminderId).toBe("rem-1");
    expect(state.reminderDraft.title).toBe("Check task links refreshed");
    expect(state.reminderDraft.message).toBe("updated reminder");
    expect(state.reminderDraft.status).toBe("sent");
    expect(state.reminderDraft.resourceId).toBe("res-2");
    expect(state.reminderDraft.remindAt).not.toBe("");
    expect(state.error).toBeNull();
  });

  it("clears missing selected reminder id but preserves the draft on list success", async () => {
    seedSelectedReminder();
    vi.mocked(listReminders).mockResolvedValueOnce([
      {
        id: "rem-2",
        title: "Replacement reminder",
        message: "new list payload",
        remindAt: "2026-04-22T09:00:00.000Z",
        status: "scheduled",
        resourceId: "res-2",
        createdAt: "2026-04-18T12:00:00.000Z",
        updatedAt: "2026-04-19T08:00:00.000Z",
      },
    ]);

    await useTaskStore.getState().loadReminders({ silent: true });

    const state = useTaskStore.getState();
    expect(state.reminders).toHaveLength(1);
    expect(state.reminders[0]?.id).toBe("rem-2");
    expect(state.selectedReminderId).toBeNull();
    expect(state.reminderDraft.title).toBe("Check task links");
    expect(state.reminderDraft.remindAt).toBe("2026-04-21T09:00");
  });

  it("sets error when todo list fails and keeps selected todo", async () => {
    seedSelectedTodo();
    vi.mocked(listTodos).mockRejectedValueOnce(new Error("mock todo list failure"));

    await useTaskStore.getState().loadTodos();

    const state = useTaskStore.getState();
    expect(state.error).toBe("mock todo list failure");
    expect(state.todos).toHaveLength(1);
    expect(state.selectedTodoId).toBe("todo-1");
  });

  it("sets error when reminder list fails and keeps selected reminder", async () => {
    seedSelectedReminder();
    vi.mocked(listReminders).mockRejectedValueOnce(new Error("mock reminder list failure"));

    await useTaskStore.getState().loadReminders();

    const state = useTaskStore.getState();
    expect(state.error).toBe("mock reminder list failure");
    expect(state.reminders).toHaveLength(1);
    expect(state.selectedReminderId).toBe("rem-1");
  });

  describe("IPC mode (window.go)", () => {
    type WailsWindow = Window & {
      go?: { desktop: { App: Record<string, (...args: unknown[]) => Promise<unknown>> } };
    };

    afterEach(() => {
      delete (window as WailsWindow).go;
    });

    it("loads todos via IPC when window.go is present", async () => {
      const getTodos = vi.fn().mockResolvedValue([
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
      ]);
      (window as WailsWindow).go = { desktop: { App: { GetTodos: getTodos } } };

      await useTaskStore.getState().loadTodos();

      expect(getTodos).toHaveBeenCalledWith(50, 0);
      expect(listTodos).not.toHaveBeenCalled();
      expect(useTaskStore.getState().todos).toHaveLength(1);
    });

    it("creates todo via IPC when window.go is present", async () => {
      const createTodoIpc = vi.fn().mockResolvedValue({
        id: "todo-1",
        title: "Draft roadmap",
        details: "link task to resource",
        status: "open",
        dueAt: "2026-04-20T10:30:00.000Z",
        resourceId: "res-1",
        createdAt: "2026-04-18T12:00:00.000Z",
        updatedAt: "2026-04-18T12:00:00.000Z",
      });
      (window as WailsWindow).go = { desktop: { App: { CreateTodo: createTodoIpc } } };

      useTaskStore.getState().updateTodoDraft("title", "Draft roadmap");
      useTaskStore.getState().updateTodoDraft("details", "link task to resource");
      useTaskStore.getState().updateTodoDraft("dueAt", "2026-04-20T10:30");
      useTaskStore.getState().updateTodoDraft("resourceId", "res-1");

      await useTaskStore.getState().addTodo();

      expect(createTodoIpc).toHaveBeenCalledWith(
        "Draft roadmap",
        "link task to resource",
        expect.any(String),
        "res-1",
      );
      expect(createTodo).not.toHaveBeenCalled();
      expect(useTaskStore.getState().todos).toHaveLength(1);
    });

    it("updates todo via IPC when window.go is present", async () => {
      seedSelectedTodo();
      const updateTodoIpc = vi.fn().mockResolvedValue({
        id: "todo-1",
        title: "Draft roadmap updated",
        details: "link task to resource",
        status: "in_progress",
        dueAt: "2026-04-20T10:30:00.000Z",
        resourceId: "res-1",
        createdAt: "2026-04-18T12:00:00.000Z",
        updatedAt: "2026-04-18T12:30:00.000Z",
      });
      (window as WailsWindow).go = { desktop: { App: { UpdateTodo: updateTodoIpc } } };

      useTaskStore.getState().updateTodoDraft("title", "Draft roadmap updated");
      useTaskStore.getState().updateTodoDraft("status", "in_progress");

      await useTaskStore.getState().updateSelectedTodo();

      expect(updateTodoIpc).toHaveBeenCalledWith(
        "todo-1",
        "Draft roadmap updated",
        "link task to resource",
        expect.any(String),
        "in_progress",
        "res-1",
      );
      expect(updateTodo).not.toHaveBeenCalled();
      expect(useTaskStore.getState().todos[0]?.title).toBe("Draft roadmap updated");
    });

    it("deletes todo via IPC when window.go is present", async () => {
      seedSelectedTodo();
      const deleteTodoIpc = vi.fn().mockResolvedValue(undefined);
      (window as WailsWindow).go = { desktop: { App: { DeleteTodo: deleteTodoIpc } } };

      await useTaskStore.getState().deleteSelectedTodo();

      expect(deleteTodoIpc).toHaveBeenCalledWith("todo-1");
      expect(deleteTodo).not.toHaveBeenCalled();
      expect(useTaskStore.getState().todos).toHaveLength(0);
    });

    it("loads reminders via IPC when window.go is present", async () => {
      const getReminders = vi.fn().mockResolvedValue([
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
      ]);
      (window as WailsWindow).go = { desktop: { App: { GetReminders: getReminders } } };

      await useTaskStore.getState().loadReminders();

      expect(getReminders).toHaveBeenCalledWith(50, 0);
      expect(listReminders).not.toHaveBeenCalled();
      expect(useTaskStore.getState().reminders).toHaveLength(1);
    });

    it("creates reminder via IPC when window.go is present", async () => {
      const createReminderIpc = vi.fn().mockResolvedValue({
        id: "rem-1",
        title: "Check task links",
        message: "verify related resource",
        remindAt: "2026-04-21T09:00:00.000Z",
        status: "scheduled",
        resourceId: "res-1",
        createdAt: "2026-04-18T12:00:00.000Z",
        updatedAt: "2026-04-18T12:00:00.000Z",
      });
      (window as WailsWindow).go = { desktop: { App: { CreateReminder: createReminderIpc } } };

      useTaskStore.getState().updateReminderDraft("title", "Check task links");
      useTaskStore.getState().updateReminderDraft("message", "verify related resource");
      useTaskStore.getState().updateReminderDraft("remindAt", "2026-04-21T09:00");
      useTaskStore.getState().updateReminderDraft("resourceId", "res-1");

      await useTaskStore.getState().addReminder();

      expect(createReminderIpc).toHaveBeenCalledWith(
        "Check task links",
        "verify related resource",
        expect.any(String),
        "res-1",
      );
      expect(createReminder).not.toHaveBeenCalled();
      expect(useTaskStore.getState().reminders).toHaveLength(1);
    });

    it("updates reminder via IPC when window.go is present", async () => {
      seedSelectedReminder();
      const updateReminderIpc = vi.fn().mockResolvedValue({
        id: "rem-1",
        title: "Check task links",
        message: "updated message",
        remindAt: "2026-04-21T09:00:00.000Z",
        status: "sent",
        resourceId: "res-1",
        createdAt: "2026-04-18T12:00:00.000Z",
        updatedAt: "2026-04-18T12:30:00.000Z",
      });
      (window as WailsWindow).go = { desktop: { App: { UpdateReminder: updateReminderIpc } } };

      useTaskStore.getState().updateReminderDraft("message", "updated message");
      useTaskStore.getState().updateReminderDraft("status", "sent");

      await useTaskStore.getState().updateSelectedReminder();

      expect(updateReminderIpc).toHaveBeenCalledWith(
        "rem-1",
        "Check task links",
        "updated message",
        expect.any(String),
        "sent",
        "res-1",
      );
      expect(updateReminder).not.toHaveBeenCalled();
      expect(useTaskStore.getState().reminders[0]?.message).toBe("updated message");
    });

    it("deletes reminder via IPC when window.go is present", async () => {
      seedSelectedReminder();
      const deleteReminderIpc = vi.fn().mockResolvedValue(undefined);
      (window as WailsWindow).go = { desktop: { App: { DeleteReminder: deleteReminderIpc } } };

      await useTaskStore.getState().deleteSelectedReminder();

      expect(deleteReminderIpc).toHaveBeenCalledWith("rem-1");
      expect(deleteReminder).not.toHaveBeenCalled();
      expect(useTaskStore.getState().reminders).toHaveLength(0);
    });
  });
});
