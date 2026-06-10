// @vitest-environment jsdom
import React from "react";

import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";

import TaskBoard from "../../../src/components/tasks/TaskBoard";
import { useResourceStore } from "../../../src/stores/useResourceStore";
import { useTaskStore } from "../../../src/stores/useTaskStore";
import type { ReminderDraft, ResourceDraft, ResourceFilters, TodoDraft, TodoItem } from "../../../src/types";

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

const defaultResourceDraft: ResourceDraft = {
  url: "",
  title: "",
  summary: "",
  categoryName: "",
};

const defaultResourceFilters: ResourceFilters = {
  query: "",
  category: "all",
  viewMode: "3d",
  showOverridesOnly: false,
};

const seedTodo: TodoItem = {
  id: "todo-1",
  title: "Ship v1",
  details: "Deploy the build",
  status: "open",
  dueAt: "2026-05-02T10:00:00Z",
  resourceId: "",
  createdAt: "2026-05-01T10:00:00Z",
  updatedAt: "2026-05-01T10:00:00Z",
};

function resetResourceStore() {
  useResourceStore.setState({
    resources: [],
    isLoading: false,
    error: null,
    selectedResourceId: null,
    filters: { ...defaultResourceFilters },
    draft: { ...defaultResourceDraft },
  });
}

function resetTaskStore() {
  useTaskStore.setState({
    todos: [],
    reminders: [],
    isLoadingTodos: false,
    isLoadingReminders: false,
    error: null,
    selectedTodoId: null,
    selectedReminderId: null,
    todoDraft: { ...defaultTodoDraft },
    reminderDraft: { ...defaultReminderDraft },
  });
}

describe("TaskBoard integration", () => {
  beforeEach(() => {
    resetTaskStore();
    resetResourceStore();
  });

  afterEach(() => {
    cleanup();
  });

  it("shows a validation error when adding a todo without a title", () => {
    render(<TaskBoard />);

    fireEvent.click(screen.getByRole("button", { name: "Add Todo" }));

    expect(screen.getByText("Todo title is required.")).toBeTruthy();
  });

  it("enables todo actions after selecting a todo", () => {
    useTaskStore.setState({ todos: [seedTodo] });

    render(<TaskBoard />);

    const todoCard = screen.getByRole("heading", { name: "Todos" }).closest("article");
    if (!todoCard) {
      throw new Error("Todo card not found");
    }

    const updateButton = within(todoCard).getByRole("button", { name: "Update" }) as HTMLButtonElement;
    const markDoneButton = within(todoCard).getByRole("button", { name: "Mark Done" }) as HTMLButtonElement;
    const deleteButton = within(todoCard).getByRole("button", { name: "Delete" }) as HTMLButtonElement;

    expect(updateButton.disabled).toBe(true);
    expect(markDoneButton.disabled).toBe(true);
    expect(deleteButton.disabled).toBe(true);

    const todoRow = screen.getByText(seedTodo.title).closest("button") as HTMLButtonElement | null;
    expect(todoRow).toBeTruthy();
    fireEvent.click(todoRow as HTMLButtonElement);

    expect(updateButton.disabled).toBe(false);
    expect(markDoneButton.disabled).toBe(false);
    expect(deleteButton.disabled).toBe(false);
  });

  it("shows a reminder time error when the title is set but time is missing", () => {
    useTaskStore.setState({
      reminderDraft: {
        ...defaultReminderDraft,
        title: "Follow up",
      },
    });

    render(<TaskBoard />);

    fireEvent.click(screen.getByRole("button", { name: "Add Reminder" }));

    expect(screen.getByText("Reminder time is required and must be valid.")).toBeTruthy();
  });
});
