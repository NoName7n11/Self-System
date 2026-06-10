import { expect, test, type Page, type Route } from "@playwright/test";

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
  failResourceCreate: boolean;
  failChatCommand: boolean;
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

async function installSyncWebSocketMock(page: Page) {
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
        // no-op: tests do not rely on outbound websocket traffic.
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

function safeHost(rawURL: string): string {
  const trimmed = rawURL.trim();
  if (trimmed === "") {
    return "";
  }

  try {
    return new URL(trimmed).host;
  } catch {
    return "";
  }
}

async function installApiMocks(
  page: Page,
  options?: {
    failResourceCreate?: boolean;
    failChatCommand?: boolean;
  },
): Promise<ApiMockState> {
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
      {
        id: "res-2",
        url: "https://example.com/research",
        host: "example.com",
        title: "Research notes",
        summary: "Background research notes.",
        category_id: "cat-2",
        category_name: "Research",
        user_override: true,
        created_at: "2026-04-20T00:00:00.000Z",
        updated_at: "2026-04-20T00:00:00.000Z",
      },
    ],
    todos: [
      {
        id: "todo-1",
        title: "Draft roadmap",
        details: "Outline the next milestone.",
        status: "open",
        due_at: "2026-05-12T09:30:00.000Z",
        resource_id: "res-1",
        created_at: "2026-05-01T09:00:00.000Z",
        updated_at: "2026-05-01T09:00:00.000Z",
      },
    ],
    reminders: [
      {
        id: "rem-1",
        title: "Sync check",
        message: "Verify sync status on launch.",
        remind_at: "2026-05-02T10:15:00.000Z",
        status: "scheduled",
        resource_id: "res-2",
        created_at: "2026-05-01T10:00:00.000Z",
        updated_at: "2026-05-01T10:00:00.000Z",
      },
    ],
    failResourceCreate: options?.failResourceCreate ?? false,
    failChatCommand: options?.failChatCommand ?? false,
  };

  await page.route("**/api/v1/**", async (route: Route) => {
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

    if (pathname === "/api/v1/resources" && method === "POST") {
      if (state.failResourceCreate) {
        await route.fulfill({
          status: 500,
          headers: {
            "content-type": "application/json",
            ...corsHeaders,
          },
          body: JSON.stringify({ error: "Failed to create resource from mock", code: "mock_resource_create_error" }),
        });
        return;
      }

      const payload = (JSON.parse(request.postData() ?? "{}") ?? {}) as Record<string, unknown>;
      const urlValue = typeof payload.url === "string" ? payload.url : "";
      const now = new Date().toISOString();
      const nextId = `res-${state.resources.length + 1}`;
      const created: ResourceItem = {
        id: nextId,
        url: urlValue,
        host: safeHost(urlValue),
        title: typeof payload.title === "string" ? payload.title : "",
        summary: typeof payload.summary === "string" ? payload.summary : "",
        category_id: `cat-${state.resources.length + 1}`,
        category_name: typeof payload.category_name === "string" ? payload.category_name : "Unsorted",
        user_override: false,
        created_at: now,
        updated_at: now,
      };

      state.resources = [created, ...state.resources];
      await fulfillEnvelope(route, created, 201);
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

    if (pathname === "/api/v1/chat/commands" && method === "POST") {
      if (state.failChatCommand) {
        await route.fulfill({
          status: 500,
          headers: {
            "content-type": "application/json",
            ...corsHeaders,
          },
          body: JSON.stringify({ error: "Chat command rejected from mock", code: "mock_chat_error" }),
        });
        return;
      }

      await fulfillEnvelope(route, { action: "list_resources", message: "Found resources." });
      return;
    }

    await fulfillEnvelope(route, []);
  });

  return state;
}

async function disableAnimations(page: Page) {
  await page.addStyleTag({
    content: "* { transition: none !important; animation: none !important; }",
  });
}

test.describe("visual snapshots", () => {
  test.use({
    viewport: { width: 1440, height: 900 },
    locale: "en-US",
    timezoneId: "UTC",
  });

  test("search layout snapshot", async ({ page }) => {
    await installApiMocks(page);

    await page.goto("/");
    await disableAnimations(page);

    await page.locator("nav.sidebar-nav").getByRole("button", { name: /Search/ }).click();
    await expect(page.getByRole("heading", { name: "Resource Ledger" })).toBeVisible();
    await expect(page.locator(".resource-row")).toHaveCount(2);

    await expect(page.locator(".app-shell")).toHaveScreenshot("search-layout.png");
  });

  test("graph layout snapshot", async ({ page }) => {
    await installApiMocks(page);

    await page.goto("/");
    await disableAnimations(page);

    await expect(page.getByRole("heading", { name: "Knowledge Graph Surface" })).toBeVisible();
    await expect(page.locator(".resource-row")).toHaveCount(2);

    const mainColumn = page.locator(".column-main");

    await expect(mainColumn).toHaveScreenshot("graph-layout.png", {
      mask: [page.locator(".graph-stage"), page.locator(".mode-toggle"), page.locator(".resource-row-meta")],
    });
  });

  test("chat layout snapshot", async ({ page }) => {
    await installApiMocks(page);

    await page.goto("/");
    await disableAnimations(page);

    await page.locator("nav.sidebar-nav").getByRole("button", { name: /Chat/ }).click();
    await expect(page.getByRole("heading", { name: "Chat Layout" })).toBeVisible();

    await expect(page.locator(".app-shell")).toHaveScreenshot("chat-layout.png");
  });

  test("tasks layout snapshot", async ({ page }) => {
    await installApiMocks(page);

    await page.goto("/");
    await disableAnimations(page);

    await page.locator("nav.sidebar-nav").getByRole("button", { name: /Tasks/ }).click();
    await expect(page.getByRole("heading", { name: "Task Operations" })).toBeVisible();

    await expect(page.locator(".app-shell")).toHaveScreenshot("tasks-layout.png");
  });

  test("settings layout snapshot", async ({ page }) => {
    await installApiMocks(page);

    await page.goto("/");
    await disableAnimations(page);

    await page.locator("nav.sidebar-nav").getByRole("button", { name: /Settings/ }).click();
    await expect(page.getByRole("heading", { name: "Runtime Settings" })).toBeVisible();

    await expect(page.locator(".app-shell")).toHaveScreenshot("settings-layout.png");
  });

  test("resource create error snapshot", async ({ page }) => {
    await installApiMocks(page, { failResourceCreate: true });

    await page.goto("/");
    await disableAnimations(page);

    const form = page.locator("section.resource-form");

    await form.getByLabel("URL").fill("https://example.com/error-target");
    await form.getByLabel("Title").fill("Create should fail");
    await form.getByLabel("Category").fill("Research");
    await form.getByLabel("Summary").fill("Snapshot of create error state.");
    await form.getByRole("button", { name: "Add As New" }).click();

    await expect(page.getByText("Failed to create resource from mock")).toBeVisible();

    await expect(page.locator(".app-shell")).toHaveScreenshot("resource-create-error.png", {
      mask: [page.locator(".graph-stage")],
    });
  });

  test("chat command error snapshot", async ({ page }) => {
    await installApiMocks(page, { failChatCommand: true });

    await page.goto("/");
    await disableAnimations(page);

    await page.locator("nav.sidebar-nav").getByRole("button", { name: /Chat/ }).click();
    await expect(page.getByRole("heading", { name: "Chat Layout" })).toBeVisible();

    const dock = page.locator("section.chat-dock");

    await dock.getByPlaceholder("Type a command...").fill("create category research");
    await dock.getByRole("button", { name: "Send" }).click();

    await expect(dock.getByText(/Command failed: Chat command rejected from mock/i)).toBeVisible();

    await expect(page.locator(".app-shell")).toHaveScreenshot("chat-error.png");
  });
});
