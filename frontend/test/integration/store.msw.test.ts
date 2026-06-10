import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it } from "vitest";

import { HttpResponse, http } from "msw";
import { setupServer } from "msw/node";

import { API_BASE_URL } from "../../src/api/client";
import { useChatStore } from "../../src/stores/useChatStore";
import { useResourceStore } from "../../src/stores/useResourceStore";
import { useTaskStore } from "../../src/stores/useTaskStore";
import type {
  ReminderDraft,
  ReminderItem,
  ResourceDraft,
  ResourceFilters,
  ResourceItem,
  TodoDraft,
  TodoItem,
} from "../../src/types";

type MockState = {
  resources: ResourceItem[];
  todos: TodoItem[];
  reminders: ReminderItem[];
  nextResourceId: number;
  nextTodoId: number;
  nextReminderId: number;
  listCalls: {
    resources: number;
    todos: number;
    reminders: number;
  };
};

const mockState: MockState = {
  resources: [],
  todos: [],
  reminders: [],
  nextResourceId: 1,
  nextTodoId: 1,
  nextReminderId: 1,
  listCalls: {
    resources: 0,
    todos: 0,
    reminders: 0,
  },
};

const server = setupServer(
  http.get(`${API_BASE_URL}/api/v1/resources`, () => {
    mockState.listCalls.resources += 1;
    return HttpResponse.json({ data: mockState.resources });
  }),
  http.post(`${API_BASE_URL}/api/v1/resources`, async ({ request }) => {
    const payload = (await request.json()) as Record<string, unknown>;
    const url = String(payload.url ?? "https://example.com/resource").trim();
    const title = String(payload.title ?? "Untitled Resource").trim();
    const summary = String(payload.summary ?? "").trim();
    const categoryName = String(payload.category_name ?? "Research").trim();

    const id = `res-${mockState.nextResourceId}`;
    mockState.nextResourceId += 1;

    const host = new URL(url).hostname.replace(/^www\./, "");
    const now = new Date().toISOString();

    const created: ResourceItem = {
      id,
      url,
      host,
      title,
      summary,
      categoryId: "cat-1",
      categoryName,
      userOverride: true,
      createdAt: now,
      updatedAt: now,
    };

    mockState.resources = [created, ...mockState.resources];

    return HttpResponse.json(
      {
        data: {
          id: created.id,
          url: created.url,
          host: created.host,
          title: created.title,
          summary: created.summary,
          category_id: created.categoryId,
          category_name: created.categoryName,
          user_override: created.userOverride,
          created_at: created.createdAt,
          updated_at: created.updatedAt,
        },
      },
      { status: 201 },
    );
  }),
  http.put(`${API_BASE_URL}/api/v1/resources/:id`, async ({ params, request }) => {
    const payload = (await request.json()) as Record<string, unknown>;
    const id = String(params.id ?? "");

    const target = mockState.resources.find((item) => item.id === id);
    if (!target) {
      return HttpResponse.json({ error: "not found" }, { status: 404 });
    }

    const url = typeof payload.url === "string" ? payload.url.trim() : target.url;
    const title = typeof payload.title === "string" ? payload.title.trim() : target.title;
    const summary = typeof payload.summary === "string" ? payload.summary.trim() : target.summary;
    const categoryName =
      typeof payload.category_name === "string" ? payload.category_name.trim() : target.categoryName;

    const updated: ResourceItem = {
      ...target,
      url,
      host: new URL(url).hostname.replace(/^www\./, ""),
      title,
      summary,
      categoryName,
      updatedAt: new Date().toISOString(),
    };

    mockState.resources = mockState.resources.map((item) => (item.id === id ? updated : item));

    return HttpResponse.json({
      data: {
        id: updated.id,
        url: updated.url,
        host: updated.host,
        title: updated.title,
        summary: updated.summary,
        category_id: updated.categoryId,
        category_name: updated.categoryName,
        user_override: updated.userOverride,
        created_at: updated.createdAt,
        updated_at: updated.updatedAt,
      },
    });
  }),
  http.delete(`${API_BASE_URL}/api/v1/resources/:id`, ({ params }) => {
    const id = String(params.id ?? "");
    mockState.resources = mockState.resources.filter((item) => item.id !== id);
    return HttpResponse.json({ message: "deleted" });
  }),
  http.get(`${API_BASE_URL}/api/v1/todos`, () => {
    mockState.listCalls.todos += 1;
    return HttpResponse.json({ data: mockState.todos });
  }),
  http.post(`${API_BASE_URL}/api/v1/todos`, async ({ request }) => {
    const payload = (await request.json()) as Record<string, unknown>;
    const title = typeof payload.title === "string" ? payload.title.trim() : "Untitled";
    const details = typeof payload.details === "string" ? payload.details.trim() : "";
    const dueAt = typeof payload.due_at === "string" ? payload.due_at.trim() : "";
    const status = typeof payload.status === "string" ? payload.status : "open";
    const resourceId = typeof payload.resource_id === "string" ? payload.resource_id.trim() : "";

    const id = `todo-${mockState.nextTodoId}`;
    mockState.nextTodoId += 1;
    const now = new Date().toISOString();

    const created: TodoItem = {
      id,
      title,
      details,
      status: status as TodoItem["status"],
      dueAt,
      resourceId,
      createdAt: now,
      updatedAt: now,
    };

    mockState.todos = [created, ...mockState.todos];

    return HttpResponse.json(
      {
        data: {
          id: created.id,
          title: created.title,
          details: created.details,
          status: created.status,
          due_at: created.dueAt,
          resource_id: created.resourceId,
          created_at: created.createdAt,
          updated_at: created.updatedAt,
        },
      },
      { status: 201 },
    );
  }),
  http.put(`${API_BASE_URL}/api/v1/todos/:id`, async ({ params, request }) => {
    const payload = (await request.json()) as Record<string, unknown>;
    const id = String(params.id ?? "");

    const target = mockState.todos.find((item) => item.id === id);
    if (!target) {
      return HttpResponse.json({ error: "not found" }, { status: 404 });
    }

    const title = typeof payload.title === "string" ? payload.title.trim() : target.title;
    const details = typeof payload.details === "string" ? payload.details.trim() : target.details;
    const status = typeof payload.status === "string" ? payload.status : target.status;
    const dueAt = typeof payload.due_at === "string" ? payload.due_at.trim() : target.dueAt;
    const resourceId = typeof payload.resource_id === "string" ? payload.resource_id.trim() : target.resourceId;

    const updated: TodoItem = {
      ...target,
      title,
      details,
      status: status as TodoItem["status"],
      dueAt,
      resourceId,
      updatedAt: new Date().toISOString(),
    };

    mockState.todos = mockState.todos.map((item) => (item.id === id ? updated : item));

    return HttpResponse.json({
      data: {
        id: updated.id,
        title: updated.title,
        details: updated.details,
        status: updated.status,
        due_at: updated.dueAt,
        resource_id: updated.resourceId,
        created_at: updated.createdAt,
        updated_at: updated.updatedAt,
      },
    });
  }),
  http.delete(`${API_BASE_URL}/api/v1/todos/:id`, ({ params }) => {
    const id = String(params.id ?? "");
    mockState.todos = mockState.todos.filter((item) => item.id !== id);
    return HttpResponse.json({ message: "deleted" });
  }),
  http.get(`${API_BASE_URL}/api/v1/reminders`, () => {
    mockState.listCalls.reminders += 1;
    return HttpResponse.json({ data: mockState.reminders });
  }),
  http.post(`${API_BASE_URL}/api/v1/reminders`, async ({ request }) => {
    const payload = (await request.json()) as Record<string, unknown>;
    const title = typeof payload.title === "string" ? payload.title.trim() : "Untitled";
    const message = typeof payload.message === "string" ? payload.message.trim() : "";
    const remindAt = typeof payload.remind_at === "string" ? payload.remind_at.trim() : "";
    const status = typeof payload.status === "string" ? payload.status : "scheduled";
    const resourceId = typeof payload.resource_id === "string" ? payload.resource_id.trim() : "";

    const id = `rem-${mockState.nextReminderId}`;
    mockState.nextReminderId += 1;
    const now = new Date().toISOString();

    const created: ReminderItem = {
      id,
      title,
      message,
      remindAt,
      status: status as ReminderItem["status"],
      resourceId,
      createdAt: now,
      updatedAt: now,
    };

    mockState.reminders = [created, ...mockState.reminders];

    return HttpResponse.json(
      {
        data: {
          id: created.id,
          title: created.title,
          message: created.message,
          remind_at: created.remindAt,
          status: created.status,
          resource_id: created.resourceId,
          created_at: created.createdAt,
          updated_at: created.updatedAt,
        },
      },
      { status: 201 },
    );
  }),
  http.put(`${API_BASE_URL}/api/v1/reminders/:id`, async ({ params, request }) => {
    const payload = (await request.json()) as Record<string, unknown>;
    const id = String(params.id ?? "");

    const target = mockState.reminders.find((item) => item.id === id);
    if (!target) {
      return HttpResponse.json({ error: "not found" }, { status: 404 });
    }

    const title = typeof payload.title === "string" ? payload.title.trim() : target.title;
    const message = typeof payload.message === "string" ? payload.message.trim() : target.message;
    const remindAt = typeof payload.remind_at === "string" ? payload.remind_at.trim() : target.remindAt;
    const status = typeof payload.status === "string" ? payload.status : target.status;
    const resourceId = typeof payload.resource_id === "string" ? payload.resource_id.trim() : target.resourceId;

    const updated: ReminderItem = {
      ...target,
      title,
      message,
      remindAt,
      status: status as ReminderItem["status"],
      resourceId,
      updatedAt: new Date().toISOString(),
    };

    mockState.reminders = mockState.reminders.map((item) => (item.id === id ? updated : item));

    return HttpResponse.json({
      data: {
        id: updated.id,
        title: updated.title,
        message: updated.message,
        remind_at: updated.remindAt,
        status: updated.status,
        resource_id: updated.resourceId,
        created_at: updated.createdAt,
        updated_at: updated.updatedAt,
      },
    });
  }),
  http.delete(`${API_BASE_URL}/api/v1/reminders/:id`, ({ params }) => {
    const id = String(params.id ?? "");
    mockState.reminders = mockState.reminders.filter((item) => item.id !== id);
    return HttpResponse.json({ message: "deleted" });
  }),
  http.post(`${API_BASE_URL}/api/v1/chat/commands`, async ({ request }) => {
    const payload = (await request.json()) as Record<string, unknown>;
    const message = String(payload.message ?? "").trim();

    if (message === "") {
      return HttpResponse.json({ error: "message is required" }, { status: 400 });
    }

    return HttpResponse.json({
      data: {
        action: "resource_created",
        message: "Resource created",
      },
    });
  }),
);

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

function resetMockState() {
  mockState.resources = [];
  mockState.todos = [];
  mockState.reminders = [];
  mockState.nextResourceId = 1;
  mockState.nextTodoId = 1;
  mockState.nextReminderId = 1;
  mockState.listCalls.resources = 0;
  mockState.listCalls.todos = 0;
  mockState.listCalls.reminders = 0;
}

function resetStores() {
  useResourceStore.setState(
    {
      resources: [],
      isLoading: false,
      error: null,
      selectedResourceId: null,
      filters: { ...defaultResourceFilters },
      draft: { ...defaultResourceDraft },
    },
  );

  useTaskStore.setState(
    {
      todos: [],
      reminders: [],
      isLoadingTodos: false,
      isLoadingReminders: false,
      error: null,
      selectedTodoId: null,
      selectedReminderId: null,
      todoDraft: { ...defaultTodoDraft },
      reminderDraft: { ...defaultReminderDraft },
    },
  );

  useChatStore.setState(
    {
      messages: [],
      isSending: false,
      error: null,
    },
  );
}

async function waitFor(condition: () => boolean, timeoutMs = 800) {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    if (condition()) {
      return;
    }
    await new Promise((resolve) => setTimeout(resolve, 25));
  }
  throw new Error("Timed out waiting for integration condition");
}

beforeAll(() => {
  server.listen({ onUnhandledRequest: "error" });
});

afterAll(() => {
  server.close();
});

afterEach(() => {
  server.resetHandlers();
});

beforeEach(() => {
  resetMockState();
  resetStores();
});

describe("resource store integration (MSW)", () => {
  it("loads and mutates resources through the API client", async () => {
    mockState.resources = [
      {
        id: "res-0",
        url: "https://example.com/initial",
        host: "example.com",
        title: "Initial Resource",
        summary: "Seeded from mock",
        categoryId: "cat-1",
        categoryName: "Research",
        userOverride: false,
        createdAt: "2026-05-01T12:00:00Z",
        updatedAt: "2026-05-01T12:00:00Z",
      },
    ];

    await useResourceStore.getState().loadResources();
    expect(useResourceStore.getState().resources).toHaveLength(1);

    useResourceStore.getState().updateDraft("url", "https://example.com/new");
    useResourceStore.getState().updateDraft("title", "New Resource");
    useResourceStore.getState().updateDraft("summary", "Created via integration test");
    useResourceStore.getState().updateDraft("categoryName", "AI");

    await useResourceStore.getState().addResource();

    const createdId = useResourceStore.getState().selectedResourceId;
    expect(createdId).not.toBeNull();
    expect(useResourceStore.getState().resources[0]?.title).toBe("New Resource");

    useResourceStore.getState().updateDraft("title", "Updated Resource");
    await useResourceStore.getState().updateSelectedResource();

    expect(useResourceStore.getState().resources[0]?.title).toBe("Updated Resource");

    await useResourceStore.getState().deleteSelectedResource();

    expect(useResourceStore.getState().resources).toHaveLength(1);
    expect(useResourceStore.getState().selectedResourceId).toBeNull();
  });

  it("surfaces resource load errors", async () => {
    server.use(
      http.get(`${API_BASE_URL}/api/v1/resources`, () =>
        HttpResponse.json({ error: "Resource list failed" }, { status: 500 }),
      ),
    );

    await useResourceStore.getState().loadResources();

    expect(useResourceStore.getState().error).toBe("Resource list failed");
    expect(useResourceStore.getState().resources).toHaveLength(0);
  });

  it("surfaces resource create errors", async () => {
    server.use(
      http.post(`${API_BASE_URL}/api/v1/resources`, () =>
        HttpResponse.json({ error: "Resource create failed" }, { status: 500 }),
      ),
    );

    useResourceStore.getState().updateDraft("url", "https://example.com/error");

    await useResourceStore.getState().addResource();

    expect(useResourceStore.getState().error).toBe("Resource create failed");
    expect(useResourceStore.getState().selectedResourceId).toBeNull();
    expect(useResourceStore.getState().resources).toHaveLength(0);
  });

  it("surfaces resource update errors", async () => {
    server.use(
      http.put(`${API_BASE_URL}/api/v1/resources/:id`, () =>
        HttpResponse.json({ error: "Resource update failed" }, { status: 500 }),
      ),
    );

    mockState.resources = [
      {
        id: "res-1",
        url: "https://example.com/seed",
        host: "example.com",
        title: "Seeded Resource",
        summary: "Seeded",
        categoryId: "cat-1",
        categoryName: "Research",
        userOverride: false,
        createdAt: "2026-05-01T12:00:00Z",
        updatedAt: "2026-05-01T12:00:00Z",
      },
    ];

    useResourceStore.setState({ resources: [...mockState.resources] });
    useResourceStore.getState().selectResource("res-1");
    useResourceStore.getState().updateDraft("title", "Updated title");

    await useResourceStore.getState().updateSelectedResource();

    expect(useResourceStore.getState().error).toBe("Resource update failed");
    expect(useResourceStore.getState().selectedResourceId).toBe("res-1");
  });
});

describe("chat mutation integration (MSW)", () => {
  it("reloads resources and tasks after chat mutation", async () => {
    mockState.resources = [
      {
        id: "res-1",
        url: "https://example.com/chat",
        host: "example.com",
        title: "Chat Resource",
        summary: "Loaded after chat mutation",
        categoryId: "cat-2",
        categoryName: "Planning",
        userOverride: false,
        createdAt: "2026-05-02T08:00:00Z",
        updatedAt: "2026-05-02T08:00:00Z",
      },
    ];

    mockState.todos = [
      {
        id: "todo-1",
        title: "Reload todo",
        details: "Loaded after chat mutation",
        status: "open",
        dueAt: "2026-05-10T09:00:00Z",
        resourceId: "res-1",
        createdAt: "2026-05-02T08:00:00Z",
        updatedAt: "2026-05-02T08:00:00Z",
      },
    ];

    mockState.reminders = [
      {
        id: "rem-1",
        title: "Reload reminder",
        message: "Loaded after chat mutation",
        remindAt: "2026-05-11T10:00:00Z",
        status: "scheduled",
        resourceId: "res-1",
        createdAt: "2026-05-02T08:00:00Z",
        updatedAt: "2026-05-02T08:00:00Z",
      },
    ];

    await useChatStore.getState().sendMessage("create resource from chat");

    await waitFor(
      () =>
        mockState.listCalls.resources > 0 &&
        mockState.listCalls.todos > 0 &&
        mockState.listCalls.reminders > 0,
    );

    expect(useResourceStore.getState().resources).toHaveLength(1);
    expect(useTaskStore.getState().todos).toHaveLength(1);
    expect(useTaskStore.getState().reminders).toHaveLength(1);
  });

  it("surfaces chat errors in the assistant message", async () => {
    server.use(
      http.post(`${API_BASE_URL}/api/v1/chat/commands`, () =>
        HttpResponse.json({ error: "Chat service offline" }, { status: 500 }),
      ),
    );

    await useChatStore.getState().sendMessage("create resource from chat");

    const messages = useChatStore.getState().messages;
    const lastMessage = messages[messages.length - 1];

    expect(useChatStore.getState().error).toBe("Chat service offline");
    expect(lastMessage?.content).toContain("Command failed: Chat service offline");
  });
});

describe("task store integration (MSW)", () => {
  it("creates, updates, and deletes todos via the API client", async () => {
    await useTaskStore.getState().loadTodos();
    expect(useTaskStore.getState().todos).toHaveLength(0);

    useTaskStore.getState().updateTodoDraft("title", "Draft todo");
    useTaskStore.getState().updateTodoDraft("details", "Integration details");
    useTaskStore.getState().updateTodoDraft("dueAt", "2026-05-12T09:30");

    await useTaskStore.getState().addTodo();

    const createdTodoId = useTaskStore.getState().selectedTodoId;
    expect(createdTodoId).not.toBeNull();
    expect(useTaskStore.getState().todos[0]?.title).toBe("Draft todo");

    useTaskStore.getState().updateTodoDraft("title", "Updated todo");
    useTaskStore.getState().updateTodoDraft("status", "done");

    await useTaskStore.getState().updateSelectedTodo();

    const updatedTodo = useTaskStore.getState().todos[0];
    expect(updatedTodo?.title).toBe("Updated todo");
    expect(updatedTodo?.status).toBe("done");

    await useTaskStore.getState().deleteSelectedTodo();

    expect(useTaskStore.getState().todos).toHaveLength(0);
    expect(useTaskStore.getState().selectedTodoId).toBeNull();
  });

  it("creates, updates, and deletes reminders via the API client", async () => {
    await useTaskStore.getState().loadReminders();
    expect(useTaskStore.getState().reminders).toHaveLength(0);

    useTaskStore.getState().updateReminderDraft("title", "Draft reminder");
    useTaskStore.getState().updateReminderDraft("message", "Integration message");
    useTaskStore.getState().updateReminderDraft("remindAt", "2026-05-13T10:15");

    await useTaskStore.getState().addReminder();

    const createdReminderId = useTaskStore.getState().selectedReminderId;
    expect(createdReminderId).not.toBeNull();
    expect(useTaskStore.getState().reminders[0]?.title).toBe("Draft reminder");

    useTaskStore.getState().updateReminderDraft("title", "Updated reminder");
    useTaskStore.getState().updateReminderDraft("status", "sent");

    await useTaskStore.getState().updateSelectedReminder();

    const updatedReminder = useTaskStore.getState().reminders[0];
    expect(updatedReminder?.title).toBe("Updated reminder");
    expect(updatedReminder?.status).toBe("sent");

    await useTaskStore.getState().deleteSelectedReminder();

    expect(useTaskStore.getState().reminders).toHaveLength(0);
    expect(useTaskStore.getState().selectedReminderId).toBeNull();
  });

  it("rejects invalid todo due date without calling the API", async () => {
    useTaskStore.getState().updateTodoDraft("title", "Invalid todo");
    useTaskStore.getState().updateTodoDraft("dueAt", "not-a-date");

    await useTaskStore.getState().addTodo();

    expect(useTaskStore.getState().error).toBe("Todo due date must be a valid date/time.");
    expect(useTaskStore.getState().todos).toHaveLength(0);
    expect(mockState.todos).toHaveLength(0);
    expect(mockState.nextTodoId).toBe(1);
  });

  it("rejects invalid reminder time without calling the API", async () => {
    useTaskStore.getState().updateReminderDraft("title", "Invalid reminder");
    useTaskStore.getState().updateReminderDraft("remindAt", "bad-time");

    await useTaskStore.getState().addReminder();

    expect(useTaskStore.getState().error).toBe("Reminder time is required and must be valid.");
    expect(useTaskStore.getState().reminders).toHaveLength(0);
    expect(mockState.reminders).toHaveLength(0);
    expect(mockState.nextReminderId).toBe(1);
  });

  it("surfaces todo create API errors", async () => {
    server.use(
      http.post(`${API_BASE_URL}/api/v1/todos`, () =>
        HttpResponse.json({ error: "Todo create failed" }, { status: 500 }),
      ),
    );

    useTaskStore.getState().updateTodoDraft("title", "Error todo");
    useTaskStore.getState().updateTodoDraft("dueAt", "2026-05-14T10:30");

    await useTaskStore.getState().addTodo();

    expect(useTaskStore.getState().error).toBe("Todo create failed");
    expect(useTaskStore.getState().todos).toHaveLength(0);
  });
});
