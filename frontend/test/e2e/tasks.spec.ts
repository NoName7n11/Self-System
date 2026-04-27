import { expect, test, type Route } from "@playwright/test";

interface ResourceItem {
  id: string;
  url: string;
  host: string;
  title: string;
  summary: string;
  category_id: string;
  category_name: string;
  user_override: boolean;
  created_at: string;
  updated_at: string;
}

interface TodoItem {
  id: string;
  title: string;
  details: string;
  status: string;
  due_at: string;
  resource_id: string;
  created_at: string;
  updated_at: string;
}

interface ReminderItem {
  id: string;
  title: string;
  message: string;
  remind_at: string;
  status: string;
  resource_id: string;
  created_at: string;
  updated_at: string;
}

interface ApiMockState {
  resources: ResourceItem[];
  todos: TodoItem[];
  reminders: ReminderItem[];
  nextTodoID: number;
  nextReminderID: number;
  abortTodoCreateNetwork: boolean;
  abortReminderCreateNetwork: boolean;
  malformedReminderCreateResponse: boolean;
  malformedTodoCreateResponse: boolean;
  failTodoCreate: boolean;
  failTodoUpdateIDs: Set<string>;
  timeoutTodoUpdateIDs: Set<string>;
  malformedTodoUpdateIDs: Set<string>;
  failTodoDeleteIDs: Set<string>;
  failReminderCreate: boolean;
  failReminderUpdateIDs: Set<string>;
  timeoutReminderUpdateIDs: Set<string>;
  malformedReminderUpdateIDs: Set<string>;
  failReminderDeleteIDs: Set<string>;
  lastTodoCreatePayload: Record<string, unknown> | null;
  lastTodoUpdatePayload: Record<string, unknown> | null;
  lastTodoDeleteID: string | null;
  lastReminderCreatePayload: Record<string, unknown> | null;
  lastReminderUpdatePayload: Record<string, unknown> | null;
  lastReminderDeleteID: string | null;
}

const corsHeaders = {
  "access-control-allow-origin": "*",
  "access-control-allow-methods": "GET,POST,PUT,DELETE,OPTIONS",
  "access-control-allow-headers": "*",
};

async function fulfillEnvelope(route: Route, data: unknown, status = 200) {
  await route.fulfill({
    status,
    headers: {
      "content-type": "application/json",
      ...corsHeaders,
    },
    body: JSON.stringify({ data }),
  });
}

async function installSyncWebSocketMock(page: Parameters<typeof test>[0]["page"]) {
  await page.addInitScript(() => {
    class MockWebSocket {
      static CONNECTING = 0;
      static OPEN = 1;
      static CLOSING = 2;
      static CLOSED = 3;

      readonly url: string;
      readonly protocol = "";
      readyState = MockWebSocket.CONNECTING;

      onopen: ((event: unknown) => void) | null = null;
      onmessage: ((event: unknown) => void) | null = null;
      onerror: ((event: unknown) => void) | null = null;
      onclose: ((event: unknown) => void) | null = null;

      constructor(url: string) {
        this.url = url;

        queueMicrotask(() => {
          this.readyState = MockWebSocket.OPEN;
          if (typeof this.onopen === "function") {
            this.onopen(new Event("open"));
          }
          if (typeof this.onmessage === "function") {
            this.onmessage(
              new MessageEvent("message", {
                data: JSON.stringify({
                  type: "sync.connected",
                  payload: { message: "mock sync connected" },
                  timestamp: new Date().toISOString(),
                }),
              }),
            );
          }
        });
      }

      send(_data: unknown) {
        // no-op: tests do not rely on websocket outbound traffic.
      }

      close() {
        if (this.readyState === MockWebSocket.CLOSED) {
          return;
        }

        this.readyState = MockWebSocket.CLOSED;
        if (typeof this.onclose === "function") {
          this.onclose({ code: 1000, reason: "mock close", wasClean: true });
        }
      }
    }

    Object.defineProperty(window, "WebSocket", {
      configurable: true,
      writable: true,
      value: MockWebSocket,
    });
  });
}

type CreateFailureMode = "backend" | "network" | "malformed";
type UpdateFailureMode = "backend" | "timeout" | "malformed";

function setTodoCreateFailure(state: ApiMockState, mode: CreateFailureMode) {
  state.failTodoCreate = mode === "backend";
  state.abortTodoCreateNetwork = mode === "network";
  state.malformedTodoCreateResponse = mode === "malformed";
}

function setReminderCreateFailure(state: ApiMockState, mode: CreateFailureMode) {
  state.failReminderCreate = mode === "backend";
  state.abortReminderCreateNetwork = mode === "network";
  state.malformedReminderCreateResponse = mode === "malformed";
}

function setTodoUpdateFailure(state: ApiMockState, todoID: string, mode: UpdateFailureMode) {
  state.failTodoUpdateIDs.delete(todoID);
  state.timeoutTodoUpdateIDs.delete(todoID);
  state.malformedTodoUpdateIDs.delete(todoID);

  if (mode === "backend") {
    state.failTodoUpdateIDs.add(todoID);
    return;
  }

  if (mode === "timeout") {
    state.timeoutTodoUpdateIDs.add(todoID);
    return;
  }

  state.malformedTodoUpdateIDs.add(todoID);
}

function setReminderUpdateFailure(state: ApiMockState, reminderID: string, mode: UpdateFailureMode) {
  state.failReminderUpdateIDs.delete(reminderID);
  state.timeoutReminderUpdateIDs.delete(reminderID);
  state.malformedReminderUpdateIDs.delete(reminderID);

  if (mode === "backend") {
    state.failReminderUpdateIDs.add(reminderID);
    return;
  }

  if (mode === "timeout") {
    state.timeoutReminderUpdateIDs.add(reminderID);
    return;
  }

  state.malformedReminderUpdateIDs.add(reminderID);
}

function setTodoDeleteFailure(state: ApiMockState, todoID: string) {
  state.failTodoDeleteIDs.add(todoID);
}

function setReminderDeleteFailure(state: ApiMockState, reminderID: string) {
  state.failReminderDeleteIDs.add(reminderID);
}

async function installApiMocks(page: Parameters<typeof test>[0]["page"]): Promise<ApiMockState> {
  await installSyncWebSocketMock(page);

  const state: ApiMockState = {
    resources: [
      {
        id: "res-1",
        url: "https://example.com/roadmap",
        host: "example.com",
        title: "Roadmap doc",
        summary: "Quarterly roadmap reference.",
        category_id: "cat-1",
        category_name: "Planning",
        user_override: false,
        created_at: "2026-04-19T00:00:00.000Z",
        updated_at: "2026-04-19T00:00:00.000Z",
      },
    ],
    todos: [],
    reminders: [],
    nextTodoID: 1,
    nextReminderID: 1,
    abortTodoCreateNetwork: false,
    abortReminderCreateNetwork: false,
    malformedReminderCreateResponse: false,
    malformedTodoCreateResponse: false,
    failTodoCreate: false,
    failTodoUpdateIDs: new Set<string>(),
    timeoutTodoUpdateIDs: new Set<string>(),
    malformedTodoUpdateIDs: new Set<string>(),
    failTodoDeleteIDs: new Set<string>(),
    failReminderCreate: false,
    failReminderUpdateIDs: new Set<string>(),
    timeoutReminderUpdateIDs: new Set<string>(),
    malformedReminderUpdateIDs: new Set<string>(),
    failReminderDeleteIDs: new Set<string>(),
    lastTodoCreatePayload: null,
    lastTodoUpdatePayload: null,
    lastTodoDeleteID: null,
    lastReminderCreatePayload: null,
    lastReminderUpdatePayload: null,
    lastReminderDeleteID: null,
  };

  await page.route("**/api/v1/**", async (route) => {
    const request = route.request();
    const method = request.method().toUpperCase();
    const url = new URL(request.url());
    const pathname = url.pathname;

    if (method === "OPTIONS") {
      await route.fulfill({
        status: 204,
        headers: corsHeaders,
      });
      return;
    }

    if (pathname === "/api/v1/resources" && method === "GET") {
      await fulfillEnvelope(route, state.resources);
      return;
    }

    if (pathname === "/api/v1/todos" && method === "GET") {
      await fulfillEnvelope(route, state.todos);
      return;
    }

    if (pathname === "/api/v1/reminders" && method === "GET") {
      await fulfillEnvelope(route, state.reminders);
      return;
    }

    if (pathname === "/api/v1/todos" && method === "POST") {
      const payload = (JSON.parse(request.postData() ?? "{}") ?? {}) as Record<string, unknown>;
      state.lastTodoCreatePayload = payload;

      if (state.failTodoCreate) {
        await route.fulfill({
          status: 500,
          headers: {
            "content-type": "application/json",
            ...corsHeaders,
          },
          body: JSON.stringify({ error: "Failed to create todo from mock", code: "mock_todo_create_error" }),
        });
        return;
      }

      if (state.abortTodoCreateNetwork) {
        await route.abort("failed");
        return;
      }

      if (state.malformedTodoCreateResponse) {
        await route.fulfill({
          status: 200,
          headers: {
            "content-type": "application/json",
            ...corsHeaders,
          },
          body: JSON.stringify({ message: "ok" }),
        });
        return;
      }

      const title = typeof payload.title === "string" ? payload.title.trim() : "Untitled";
      const details = typeof payload.details === "string" ? payload.details : "";
      const dueAt = typeof payload.due_at === "string" ? payload.due_at : "";
      const resourceID = typeof payload.resource_id === "string" ? payload.resource_id : "";

      const now = new Date().toISOString();
      const created: TodoItem = {
        id: `todo-${state.nextTodoID}`,
        title,
        details,
        status: "open",
        due_at: dueAt,
        resource_id: resourceID,
        created_at: now,
        updated_at: now,
      };

      state.nextTodoID += 1;
      state.todos = [created, ...state.todos];

      await fulfillEnvelope(route, created, 201);
      return;
    }

    if (pathname.startsWith("/api/v1/todos/") && method === "PUT") {
      const payload = (JSON.parse(request.postData() ?? "{}") ?? {}) as Record<string, unknown>;
      state.lastTodoUpdatePayload = payload;

      const todoID = pathname.split("/").pop() ?? "";
      const target = state.todos.find((item) => item.id === todoID);
      if (!target) {
        await route.fulfill({
          status: 404,
          headers: {
            "content-type": "application/json",
            ...corsHeaders,
          },
          body: JSON.stringify({ error: "not found", code: "not_found" }),
        });
        return;
      }

      if (state.failTodoUpdateIDs.has(todoID)) {
        await route.fulfill({
          status: 500,
          headers: {
            "content-type": "application/json",
            ...corsHeaders,
          },
          body: JSON.stringify({ error: "Failed to update todo from mock", code: "mock_todo_update_error" }),
        });
        return;
      }

      if (state.timeoutTodoUpdateIDs.has(todoID)) {
        await route.fulfill({
          status: 504,
          headers: {
            "content-type": "application/json",
            ...corsHeaders,
          },
          body: JSON.stringify({ error: "Todo update timed out from mock", code: "mock_todo_timeout" }),
        });
        return;
      }

      if (state.malformedTodoUpdateIDs.has(todoID)) {
        await route.fulfill({
          status: 200,
          headers: {
            "content-type": "application/json",
            ...corsHeaders,
          },
          body: JSON.stringify({ message: "ok" }),
        });
        return;
      }

      const updated: TodoItem = {
        ...target,
        title: typeof payload.title === "string" ? payload.title : target.title,
        details: typeof payload.details === "string" ? payload.details : target.details,
        due_at: typeof payload.due_at === "string" ? payload.due_at : target.due_at,
        status: typeof payload.status === "string" ? payload.status : target.status,
        resource_id: typeof payload.resource_id === "string" ? payload.resource_id : target.resource_id,
        updated_at: new Date().toISOString(),
      };

      state.todos = state.todos.map((item) => (item.id === updated.id ? updated : item));
      await fulfillEnvelope(route, updated);
      return;
    }

    if (pathname.startsWith("/api/v1/todos/") && method === "DELETE") {
      const todoID = pathname.split("/").pop() ?? "";
      state.lastTodoDeleteID = todoID;

      if (state.failTodoDeleteIDs.has(todoID)) {
        await route.fulfill({
          status: 500,
          headers: {
            "content-type": "application/json",
            ...corsHeaders,
          },
          body: JSON.stringify({ error: "Failed to delete todo from mock", code: "mock_todo_delete_error" }),
        });
        return;
      }

      state.todos = state.todos.filter((item) => item.id !== todoID);

      await route.fulfill({
        status: 200,
        headers: {
          "content-type": "application/json",
          ...corsHeaders,
        },
        body: JSON.stringify({ message: "deleted" }),
      });
      return;
    }

    if (pathname === "/api/v1/reminders" && method === "POST") {
      const payload = (JSON.parse(request.postData() ?? "{}") ?? {}) as Record<string, unknown>;
      state.lastReminderCreatePayload = payload;

      if (state.failReminderCreate) {
        await route.fulfill({
          status: 500,
          headers: {
            "content-type": "application/json",
            ...corsHeaders,
          },
          body: JSON.stringify({ error: "Failed to create reminder from mock", code: "mock_reminder_create_error" }),
        });
        return;
      }

      if (state.abortReminderCreateNetwork) {
        await route.abort("failed");
        return;
      }

      if (state.malformedReminderCreateResponse) {
        await route.fulfill({
          status: 200,
          headers: {
            "content-type": "application/json",
            ...corsHeaders,
          },
          body: JSON.stringify({ message: "ok" }),
        });
        return;
      }

      const title = typeof payload.title === "string" ? payload.title.trim() : "Untitled";
      const message = typeof payload.message === "string" ? payload.message : "";
      const remindAt = typeof payload.remind_at === "string" ? payload.remind_at : "";
      const resourceID = typeof payload.resource_id === "string" ? payload.resource_id : "";

      const now = new Date().toISOString();
      const created: ReminderItem = {
        id: `rem-${state.nextReminderID}`,
        title,
        message,
        remind_at: remindAt,
        status: "scheduled",
        resource_id: resourceID,
        created_at: now,
        updated_at: now,
      };

      state.nextReminderID += 1;
      state.reminders = [created, ...state.reminders];

      await fulfillEnvelope(route, created, 201);
      return;
    }

    if (pathname.startsWith("/api/v1/reminders/") && method === "PUT") {
      const payload = (JSON.parse(request.postData() ?? "{}") ?? {}) as Record<string, unknown>;
      state.lastReminderUpdatePayload = payload;

      const reminderID = pathname.split("/").pop() ?? "";
      const target = state.reminders.find((item) => item.id === reminderID);
      if (!target) {
        await route.fulfill({
          status: 404,
          headers: {
            "content-type": "application/json",
            ...corsHeaders,
          },
          body: JSON.stringify({ error: "not found", code: "not_found" }),
        });
        return;
      }

      if (state.failReminderUpdateIDs.has(reminderID)) {
        await route.fulfill({
          status: 500,
          headers: {
            "content-type": "application/json",
            ...corsHeaders,
          },
          body: JSON.stringify({ error: "Failed to update reminder from mock", code: "mock_reminder_update_error" }),
        });
        return;
      }

      if (state.timeoutReminderUpdateIDs.has(reminderID)) {
        await route.fulfill({
          status: 504,
          headers: {
            "content-type": "application/json",
            ...corsHeaders,
          },
          body: JSON.stringify({ error: "Reminder update timed out from mock", code: "mock_reminder_timeout" }),
        });
        return;
      }

      if (state.malformedReminderUpdateIDs.has(reminderID)) {
        await route.fulfill({
          status: 200,
          headers: {
            "content-type": "application/json",
            ...corsHeaders,
          },
          body: JSON.stringify({ message: "ok" }),
        });
        return;
      }

      const updated: ReminderItem = {
        ...target,
        title: typeof payload.title === "string" ? payload.title : target.title,
        message: typeof payload.message === "string" ? payload.message : target.message,
        remind_at: typeof payload.remind_at === "string" ? payload.remind_at : target.remind_at,
        status: typeof payload.status === "string" ? payload.status : target.status,
        resource_id: typeof payload.resource_id === "string" ? payload.resource_id : target.resource_id,
        updated_at: new Date().toISOString(),
      };

      state.reminders = state.reminders.map((item) => (item.id === updated.id ? updated : item));
      await fulfillEnvelope(route, updated);
      return;
    }

    if (pathname.startsWith("/api/v1/reminders/") && method === "DELETE") {
      const reminderID = pathname.split("/").pop() ?? "";
      state.lastReminderDeleteID = reminderID;

      if (state.failReminderDeleteIDs.has(reminderID)) {
        await route.fulfill({
          status: 500,
          headers: {
            "content-type": "application/json",
            ...corsHeaders,
          },
          body: JSON.stringify({ error: "Failed to delete reminder from mock", code: "mock_reminder_delete_error" }),
        });
        return;
      }

      state.reminders = state.reminders.filter((item) => item.id !== reminderID);

      await route.fulfill({
        status: 200,
        headers: {
          "content-type": "application/json",
          ...corsHeaders,
        },
        body: JSON.stringify({ message: "deleted" }),
      });
      return;
    }

    await fulfillEnvelope(route, []);
  });

  return state;
}

test("creates todo with linked resource from tasks section", async ({ page }) => {
  const state = await installApiMocks(page);

  await page.goto("/");

  await page.getByRole("button", { name: "Tasks" }).click();
  await expect(page.getByRole("heading", { name: "Task Operations" })).toBeVisible();

  const todoCard = page.locator("article.task-card", { hasText: "Todos" }).first();

  await todoCard.getByLabel("Title").fill("Draft roadmap");
  await todoCard.getByLabel("Details").fill("Link this todo to the roadmap resource.");
  await todoCard.getByLabel("Due at").fill("2026-05-12T09:30");
  await todoCard.getByLabel("Linked resource").selectOption("res-1");

  await todoCard.getByRole("button", { name: "Add Todo" }).click();

  await expect.poll(() => state.lastTodoCreatePayload?.resource_id).toBe("res-1");
  await expect(todoCard.getByText("Draft roadmap")).toBeVisible();
  await expect(todoCard.getByText("Resource: res-1")).toBeVisible();
  await expect(todoCard.getByText(/1 open/i)).toBeVisible();
});

test("shows todo title validation error without sending create request", async ({ page }) => {
  const state = await installApiMocks(page);

  await page.goto("/");

  await page.getByRole("button", { name: "Tasks" }).click();
  await expect(page.getByRole("heading", { name: "Task Operations" })).toBeVisible();

  const todoCard = page.locator("article.task-card", { hasText: "Todos" }).first();

  await todoCard.getByLabel("Details").fill("Validation path test");
  await todoCard.getByLabel("Due at").fill("2026-05-13T09:00");
  await todoCard.getByRole("button", { name: "Add Todo" }).click();

  await expect(page.getByText("Todo title is required.")).toBeVisible();
  await expect.poll(() => state.lastTodoCreatePayload).toBeNull();
  await expect(todoCard.getByText("No todos yet.")).toBeVisible();
});

test("shows todo create error and keeps todo list empty", async ({ page }) => {
  const state = await installApiMocks(page);
  setTodoCreateFailure(state, "backend");

  await page.goto("/");

  await page.getByRole("button", { name: "Tasks" }).click();
  await expect(page.getByRole("heading", { name: "Task Operations" })).toBeVisible();

  const todoCard = page.locator("article.task-card", { hasText: "Todos" }).first();

  await todoCard.getByLabel("Title").fill("Todo create should fail");
  await todoCard.getByLabel("Details").fill("Create error path test");
  await todoCard.getByLabel("Due at").fill("2026-05-14T07:30");
  await todoCard.getByRole("button", { name: "Add Todo" }).click();

  await expect.poll(() => state.lastTodoCreatePayload?.title).toBe("Todo create should fail");
  await expect(page.getByText("Failed to create todo from mock")).toBeVisible();
  await expect(todoCard.getByText("No todos yet.")).toBeVisible();
  await expect(todoCard.getByText("Todo create should fail")).toHaveCount(0);
});

test("shows todo create envelope error and keeps todo list empty", async ({ page }) => {
  const state = await installApiMocks(page);
  setTodoCreateFailure(state, "malformed");

  await page.goto("/");

  await page.getByRole("button", { name: "Tasks" }).click();
  await expect(page.getByRole("heading", { name: "Task Operations" })).toBeVisible();

  const todoCard = page.locator("article.task-card", { hasText: "Todos" }).first();

  await todoCard.getByLabel("Title").fill("Todo malformed response should fail");
  await todoCard.getByLabel("Details").fill("Malformed data envelope path");
  await todoCard.getByLabel("Due at").fill("2026-05-14T08:15");
  await todoCard.getByRole("button", { name: "Add Todo" }).click();

  await expect.poll(() => state.lastTodoCreatePayload?.title).toBe("Todo malformed response should fail");
  await expect(page.getByText("API response did not include data")).toBeVisible();
  await expect(todoCard.getByText("No todos yet.")).toBeVisible();
  await expect(todoCard.getByText("Todo malformed response should fail")).toHaveCount(0);
});

test("shows todo create network error and keeps todo list empty", async ({ page }) => {
  const state = await installApiMocks(page);
  setTodoCreateFailure(state, "network");

  await page.goto("/");

  await page.getByRole("button", { name: "Tasks" }).click();
  await expect(page.getByRole("heading", { name: "Task Operations" })).toBeVisible();

  const todoCard = page.locator("article.task-card", { hasText: "Todos" }).first();

  await todoCard.getByLabel("Title").fill("Todo network failure should fail");
  await todoCard.getByLabel("Details").fill("Transport failure path");
  await todoCard.getByLabel("Due at").fill("2026-05-15T08:00");
  await todoCard.getByRole("button", { name: "Add Todo" }).click();

  await expect.poll(() => state.lastTodoCreatePayload?.title).toBe("Todo network failure should fail");
  await expect(page.getByText(/Failed to fetch|NetworkError|Load failed/i)).toBeVisible();
  await expect(todoCard.getByText("No todos yet.")).toBeVisible();
  await expect(todoCard.getByText("Todo network failure should fail")).toHaveCount(0);
});

test("updates todo status and marks it done in tasks section", async ({ page }) => {
  const state = await installApiMocks(page);

  await page.goto("/");

  await page.getByRole("button", { name: "Tasks" }).click();
  await expect(page.getByRole("heading", { name: "Task Operations" })).toBeVisible();

  const todoCard = page.locator("article.task-card", { hasText: "Todos" }).first();

  await todoCard.getByLabel("Title").fill("Roadmap checkpoint");
  await todoCard.getByLabel("Details").fill("Initial todo details.");
  await todoCard.getByLabel("Due at").fill("2026-05-18T10:00");
  await todoCard.getByLabel("Linked resource").selectOption("res-1");
  await todoCard.getByRole("button", { name: "Add Todo" }).click();

  await expect.poll(() => state.lastTodoCreatePayload?.resource_id).toBe("res-1");
  await expect(todoCard.getByText("Roadmap checkpoint")).toBeVisible();

  await todoCard.locator(".task-row").first().click();

  await todoCard.locator('label:has-text("Status") select').first().selectOption("in_progress");
  await todoCard.locator(".form-actions").getByRole("button", { name: "Update" }).click();

  await expect.poll(() => state.lastTodoUpdatePayload?.status).toBe("in_progress");
  await expect(todoCard.locator(".task-row .resource-chip").first()).toHaveText("in progress");

  await todoCard.locator(".form-actions").getByRole("button", { name: "Mark Done" }).click();

  await expect.poll(() => state.lastTodoUpdatePayload?.status).toBe("done");
  await expect(todoCard.locator(".task-row .resource-chip").first()).toHaveText("done");
});

test("shows todo update error and keeps previous todo state", async ({ page }) => {
  const state = await installApiMocks(page);

  await page.goto("/");

  await page.getByRole("button", { name: "Tasks" }).click();
  await expect(page.getByRole("heading", { name: "Task Operations" })).toBeVisible();

  const todoCard = page.locator("article.task-card", { hasText: "Todos" }).first();

  await todoCard.getByLabel("Title").fill("Roadmap review");
  await todoCard.getByLabel("Details").fill("Initial details.");
  await todoCard.getByLabel("Due at").fill("2026-05-20T08:00");
  await todoCard.getByRole("button", { name: "Add Todo" }).click();

  await expect(todoCard.getByText("Roadmap review")).toBeVisible();

  setTodoUpdateFailure(state, "todo-1", "backend");

  await todoCard.getByLabel("Title").fill("Roadmap review should fail");
  await todoCard.locator(".form-actions").getByRole("button", { name: "Update" }).click();

  await expect.poll(() => state.lastTodoUpdatePayload !== null).toBe(true);
  await expect(page.getByText("Failed to update todo from mock")).toBeVisible();
  await expect(todoCard.getByText("Roadmap review")).toBeVisible();
});

test("shows todo update timeout error and keeps previous todo state", async ({ page }) => {
  const state = await installApiMocks(page);

  await page.goto("/");

  await page.getByRole("button", { name: "Tasks" }).click();
  await expect(page.getByRole("heading", { name: "Task Operations" })).toBeVisible();

  const todoCard = page.locator("article.task-card", { hasText: "Todos" }).first();

  await todoCard.getByLabel("Title").fill("Roadmap review timeout");
  await todoCard.getByLabel("Details").fill("Initial details.");
  await todoCard.getByLabel("Due at").fill("2026-05-20T08:05");
  await todoCard.getByRole("button", { name: "Add Todo" }).click();

  await expect(todoCard.getByText("Roadmap review timeout")).toBeVisible();

  setTodoUpdateFailure(state, "todo-1", "timeout");

  await todoCard.getByLabel("Title").fill("Roadmap review timeout should fail");
  await todoCard.locator(".form-actions").getByRole("button", { name: "Update" }).click();

  await expect.poll(() => state.lastTodoUpdatePayload !== null).toBe(true);
  await expect(page.getByText("Todo update timed out from mock")).toBeVisible();
  await expect(todoCard.getByText("Roadmap review timeout")).toBeVisible();
});

test("shows todo update envelope error and keeps previous todo state", async ({ page }) => {
  const state = await installApiMocks(page);

  await page.goto("/");

  await page.getByRole("button", { name: "Tasks" }).click();
  await expect(page.getByRole("heading", { name: "Task Operations" })).toBeVisible();

  const todoCard = page.locator("article.task-card", { hasText: "Todos" }).first();

  await todoCard.getByLabel("Title").fill("Roadmap review malformed");
  await todoCard.getByLabel("Details").fill("Initial details.");
  await todoCard.getByLabel("Due at").fill("2026-05-20T08:10");
  await todoCard.getByRole("button", { name: "Add Todo" }).click();

  await expect(todoCard.getByText("Roadmap review malformed")).toBeVisible();

  setTodoUpdateFailure(state, "todo-1", "malformed");

  await todoCard.getByLabel("Title").fill("Roadmap review malformed should fail");
  await todoCard.locator(".form-actions").getByRole("button", { name: "Update" }).click();

  await expect.poll(() => state.lastTodoUpdatePayload !== null).toBe(true);
  await expect(page.getByText("API response did not include data")).toBeVisible();
  await expect(todoCard.getByText("Roadmap review malformed")).toBeVisible();
});

test("shows todo mark done error and keeps todo status unchanged", async ({ page }) => {
  const state = await installApiMocks(page);

  await page.goto("/");

  await page.getByRole("button", { name: "Tasks" }).click();
  await expect(page.getByRole("heading", { name: "Task Operations" })).toBeVisible();

  const todoCard = page.locator("article.task-card", { hasText: "Todos" }).first();

  await todoCard.getByLabel("Title").fill("Mark done failure todo");
  await todoCard.getByLabel("Details").fill("Todo status should stay open when mark done fails.");
  await todoCard.getByLabel("Due at").fill("2026-05-21T10:00");
  await todoCard.getByRole("button", { name: "Add Todo" }).click();

  await expect(todoCard.getByText("Mark done failure todo")).toBeVisible();
  await expect(todoCard.locator(".task-row .resource-chip").first()).toHaveText("open");

  setTodoUpdateFailure(state, "todo-1", "backend");

  await todoCard.locator(".form-actions").getByRole("button", { name: "Mark Done" }).click();

  await expect.poll(() => state.lastTodoUpdatePayload?.status).toBe("done");
  await expect(page.getByText("Failed to update todo from mock")).toBeVisible();
  await expect(todoCard.locator(".task-row .resource-chip").first()).toHaveText("open");
  await expect(todoCard.getByText(/1 open/i)).toBeVisible();
});

test("shows todo delete error and keeps todo row", async ({ page }) => {
  const state = await installApiMocks(page);

  await page.goto("/");

  await page.getByRole("button", { name: "Tasks" }).click();
  await expect(page.getByRole("heading", { name: "Task Operations" })).toBeVisible();

  const todoCard = page.locator("article.task-card", { hasText: "Todos" }).first();

  await todoCard.getByLabel("Title").fill("Todo delete failure check");
  await todoCard.getByLabel("Details").fill("Todo should remain visible after failed delete.");
  await todoCard.getByLabel("Due at").fill("2026-05-22T09:10");
  await todoCard.getByRole("button", { name: "Add Todo" }).click();

  await expect(todoCard.getByText("Todo delete failure check")).toBeVisible();

  setTodoDeleteFailure(state, "todo-1");

  await todoCard.locator(".form-actions").getByRole("button", { name: "Delete" }).click();

  await expect.poll(() => state.lastTodoDeleteID).toBe("todo-1");
  await expect(page.getByText("Failed to delete todo from mock")).toBeVisible();
  await expect.poll(() => state.todos.length).toBe(1);
  await expect(todoCard.getByText("Todo delete failure check")).toBeVisible();
});

test("creates reminder and marks it sent in tasks section", async ({ page }) => {
  const state = await installApiMocks(page);

  await page.goto("/");

  await page.getByRole("button", { name: "Tasks" }).click();
  await expect(page.getByRole("heading", { name: "Task Operations" })).toBeVisible();

  const reminderCard = page.locator("article.task-card", { hasText: "Reminders" }).first();

  await reminderCard.getByLabel("Title").fill("Release checkpoint");
  await reminderCard.getByLabel("Message").fill("Verify linked reminder status flow.");
  await reminderCard.getByLabel("Remind at").fill("2026-05-15T14:00");
  await reminderCard.getByLabel("Linked resource").selectOption("res-1");

  await reminderCard.getByRole("button", { name: "Add Reminder" }).click();

  await expect.poll(() => state.lastReminderCreatePayload?.resource_id).toBe("res-1");
  await expect(reminderCard.getByText("Release checkpoint")).toBeVisible();
  await expect(reminderCard.locator(".task-row .resource-chip").first()).toHaveText("scheduled");

  await reminderCard.locator(".form-actions").getByRole("button", { name: "Mark Sent" }).click();

  await expect.poll(() => state.lastReminderUpdatePayload?.status).toBe("sent");
  await expect.poll(() => state.lastReminderUpdatePayload?.resource_id).toBe("res-1");
  await expect(reminderCard.locator(".task-row .resource-chip").first()).toHaveText("sent");
});

test("shows reminder time validation error without sending create request", async ({ page }) => {
  const state = await installApiMocks(page);

  await page.goto("/");

  await page.getByRole("button", { name: "Tasks" }).click();
  await expect(page.getByRole("heading", { name: "Task Operations" })).toBeVisible();

  const reminderCard = page.locator("article.task-card", { hasText: "Reminders" }).first();

  await reminderCard.getByLabel("Title").fill("Reminder validation should fail");
  await reminderCard.getByLabel("Message").fill("Missing remind-at validation path");
  await reminderCard.getByRole("button", { name: "Add Reminder" }).click();

  await expect(page.getByText("Reminder time is required and must be valid.")).toBeVisible();
  await expect.poll(() => state.lastReminderCreatePayload).toBeNull();
  await expect(reminderCard.getByText("No reminders yet.")).toBeVisible();
});

test("shows reminder create error and keeps reminder list empty", async ({ page }) => {
  const state = await installApiMocks(page);
  setReminderCreateFailure(state, "backend");

  await page.goto("/");

  await page.getByRole("button", { name: "Tasks" }).click();
  await expect(page.getByRole("heading", { name: "Task Operations" })).toBeVisible();

  const reminderCard = page.locator("article.task-card", { hasText: "Reminders" }).first();

  await reminderCard.getByLabel("Title").fill("Reminder create should fail");
  await reminderCard.getByLabel("Message").fill("Create error path test");
  await reminderCard.getByLabel("Remind at").fill("2026-05-17T12:40");
  await reminderCard.getByRole("button", { name: "Add Reminder" }).click();

  await expect.poll(() => state.lastReminderCreatePayload?.title).toBe("Reminder create should fail");
  await expect(page.getByText("Failed to create reminder from mock")).toBeVisible();
  await expect(reminderCard.getByText("No reminders yet.")).toBeVisible();
  await expect(reminderCard.getByText("Reminder create should fail")).toHaveCount(0);
});

test("shows reminder create network error and keeps reminder list empty", async ({ page }) => {
  const state = await installApiMocks(page);
  setReminderCreateFailure(state, "network");

  await page.goto("/");

  await page.getByRole("button", { name: "Tasks" }).click();
  await expect(page.getByRole("heading", { name: "Task Operations" })).toBeVisible();

  const reminderCard = page.locator("article.task-card", { hasText: "Reminders" }).first();

  await reminderCard.getByLabel("Title").fill("Reminder network failure should fail");
  await reminderCard.getByLabel("Message").fill("Transport failure path");
  await reminderCard.getByLabel("Remind at").fill("2026-05-17T12:55");
  await reminderCard.getByRole("button", { name: "Add Reminder" }).click();

  await expect.poll(() => state.lastReminderCreatePayload?.title).toBe("Reminder network failure should fail");
  await expect(page.getByText(/Failed to fetch|NetworkError|Load failed/i)).toBeVisible();
  await expect(reminderCard.getByText("No reminders yet.")).toBeVisible();
  await expect(reminderCard.getByText("Reminder network failure should fail")).toHaveCount(0);
});

test("shows reminder create envelope error and keeps reminder list empty", async ({ page }) => {
  const state = await installApiMocks(page);
  setReminderCreateFailure(state, "malformed");

  await page.goto("/");

  await page.getByRole("button", { name: "Tasks" }).click();
  await expect(page.getByRole("heading", { name: "Task Operations" })).toBeVisible();

  const reminderCard = page.locator("article.task-card", { hasText: "Reminders" }).first();

  await reminderCard.getByLabel("Title").fill("Reminder malformed response should fail");
  await reminderCard.getByLabel("Message").fill("Malformed data envelope path");
  await reminderCard.getByLabel("Remind at").fill("2026-05-17T13:05");
  await reminderCard.getByRole("button", { name: "Add Reminder" }).click();

  await expect.poll(() => state.lastReminderCreatePayload?.title).toBe("Reminder malformed response should fail");
  await expect(page.getByText("API response did not include data")).toBeVisible();
  await expect(reminderCard.getByText("No reminders yet.")).toBeVisible();
  await expect(reminderCard.getByText("Reminder malformed response should fail")).toHaveCount(0);
});

test("shows reminder mark sent error and keeps reminder status unchanged", async ({ page }) => {
  const state = await installApiMocks(page);

  await page.goto("/");

  await page.getByRole("button", { name: "Tasks" }).click();
  await expect(page.getByRole("heading", { name: "Task Operations" })).toBeVisible();

  const reminderCard = page.locator("article.task-card", { hasText: "Reminders" }).first();

  await reminderCard.getByLabel("Title").fill("Mark sent failure reminder");
  await reminderCard.getByLabel("Message").fill("Reminder status should stay scheduled when mark sent fails.");
  await reminderCard.getByLabel("Remind at").fill("2026-05-22T16:10");
  await reminderCard.getByRole("button", { name: "Add Reminder" }).click();

  await expect(reminderCard.getByText("Mark sent failure reminder")).toBeVisible();
  await expect(reminderCard.locator(".task-row .resource-chip").first()).toHaveText("scheduled");

  setReminderUpdateFailure(state, "rem-1", "backend");

  await reminderCard.locator(".form-actions").getByRole("button", { name: "Mark Sent" }).click();

  await expect.poll(() => state.lastReminderUpdatePayload?.status).toBe("sent");
  await expect(page.getByText("Failed to update reminder from mock")).toBeVisible();
  await expect.poll(() => state.reminders[0]?.status).toBe("scheduled");
  await expect(reminderCard.locator(".task-row .resource-chip").first()).toHaveText("scheduled");
});

test("shows reminder mark sent envelope error and keeps reminder status unchanged", async ({ page }) => {
  const state = await installApiMocks(page);

  await page.goto("/");

  await page.getByRole("button", { name: "Tasks" }).click();
  await expect(page.getByRole("heading", { name: "Task Operations" })).toBeVisible();

  const reminderCard = page.locator("article.task-card", { hasText: "Reminders" }).first();

  await reminderCard.getByLabel("Title").fill("Mark sent malformed response reminder");
  await reminderCard.getByLabel("Message").fill("Reminder should stay scheduled when update payload is malformed.");
  await reminderCard.getByLabel("Remind at").fill("2026-05-23T08:20");
  await reminderCard.getByRole("button", { name: "Add Reminder" }).click();

  await expect(reminderCard.getByText("Mark sent malformed response reminder")).toBeVisible();
  await expect(reminderCard.locator(".task-row .resource-chip").first()).toHaveText("scheduled");

  setReminderUpdateFailure(state, "rem-1", "malformed");

  await reminderCard.locator(".form-actions").getByRole("button", { name: "Mark Sent" }).click();

  await expect.poll(() => state.lastReminderUpdatePayload?.status).toBe("sent");
  await expect(page.getByText("API response did not include data")).toBeVisible();
  await expect.poll(() => state.reminders[0]?.status).toBe("scheduled");
  await expect(reminderCard.locator(".task-row .resource-chip").first()).toHaveText("scheduled");
});

test("shows reminder mark sent timeout error and keeps reminder status unchanged", async ({ page }) => {
  const state = await installApiMocks(page);

  await page.goto("/");

  await page.getByRole("button", { name: "Tasks" }).click();
  await expect(page.getByRole("heading", { name: "Task Operations" })).toBeVisible();

  const reminderCard = page.locator("article.task-card", { hasText: "Reminders" }).first();

  await reminderCard.getByLabel("Title").fill("Mark sent timeout reminder");
  await reminderCard.getByLabel("Message").fill("Reminder should stay scheduled when update times out.");
  await reminderCard.getByLabel("Remind at").fill("2026-05-24T07:40");
  await reminderCard.getByRole("button", { name: "Add Reminder" }).click();

  await expect(reminderCard.getByText("Mark sent timeout reminder")).toBeVisible();
  await expect(reminderCard.locator(".task-row .resource-chip").first()).toHaveText("scheduled");

  setReminderUpdateFailure(state, "rem-1", "timeout");

  await reminderCard.locator(".form-actions").getByRole("button", { name: "Mark Sent" }).click();

  await expect.poll(() => state.lastReminderUpdatePayload?.status).toBe("sent");
  await expect(page.getByText("Reminder update timed out from mock")).toBeVisible();
  await expect.poll(() => state.reminders[0]?.status).toBe("scheduled");
  await expect(reminderCard.locator(".task-row .resource-chip").first()).toHaveText("scheduled");
});

test("shows reminder delete error and keeps reminder row", async ({ page }) => {
  const state = await installApiMocks(page);

  await page.goto("/");

  await page.getByRole("button", { name: "Tasks" }).click();
  await expect(page.getByRole("heading", { name: "Task Operations" })).toBeVisible();

  const reminderCard = page.locator("article.task-card", { hasText: "Reminders" }).first();

  await reminderCard.getByLabel("Title").fill("Reminder failure check");
  await reminderCard.getByLabel("Message").fill("Reminder should stay visible after failed delete.");
  await reminderCard.getByLabel("Remind at").fill("2026-05-21T10:30");
  await reminderCard.getByRole("button", { name: "Add Reminder" }).click();

  await expect(reminderCard.getByText("Reminder failure check")).toBeVisible();

  setReminderDeleteFailure(state, "rem-1");

  await reminderCard.locator(".form-actions").getByRole("button", { name: "Delete" }).click();

  await expect.poll(() => state.lastReminderDeleteID).toBe("rem-1");
  await expect(page.getByText("Failed to delete reminder from mock")).toBeVisible();
  await expect.poll(() => state.reminders.length).toBe(1);
  await expect(reminderCard.getByText("Reminder failure check")).toBeVisible();
});

test("deletes reminder from tasks section", async ({ page }) => {
  const state = await installApiMocks(page);

  await page.goto("/");

  await page.getByRole("button", { name: "Tasks" }).click();
  await expect(page.getByRole("heading", { name: "Task Operations" })).toBeVisible();

  const reminderCard = page.locator("article.task-card", { hasText: "Reminders" }).first();

  await reminderCard.getByLabel("Title").fill("Release checkpoint cleanup");
  await reminderCard.getByLabel("Message").fill("Remove reminder once completed.");
  await reminderCard.getByLabel("Remind at").fill("2026-05-19T11:45");
  await reminderCard.getByLabel("Linked resource").selectOption("res-1");
  await reminderCard.getByRole("button", { name: "Add Reminder" }).click();

  await expect.poll(() => state.lastReminderCreatePayload?.resource_id).toBe("res-1");
  await expect(reminderCard.getByText("Release checkpoint cleanup")).toBeVisible();

  await reminderCard.locator(".form-actions").getByRole("button", { name: "Delete" }).click();

  await expect.poll(() => state.lastReminderDeleteID).toBe("rem-1");
  await expect.poll(() => state.reminders.length).toBe(0);
  await expect(reminderCard.getByText("Release checkpoint cleanup")).toHaveCount(0);
  await expect(reminderCard.getByText("No reminders yet.")).toBeVisible();
});
