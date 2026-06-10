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

async function installApiMocks(page: Page): Promise<ApiMockState> {
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

    if (pathname === "/api/v1/todos" && method === "GET") {
      await fulfillEnvelope(route, state.todos);
      return;
    }

    if (pathname === "/api/v1/reminders" && method === "GET") {
      await fulfillEnvelope(route, state.reminders);
      return;
    }

    await fulfillEnvelope(route, []);
  });

  return state;
}

test("filters resources via the topbar search", async ({ page }) => {
  await installApiMocks(page);

  await page.goto("/");

  await page.locator("nav.sidebar-nav").getByRole("button", { name: /Search/ }).click();
  await expect(page.getByRole("heading", { name: "Resource Ledger" })).toBeVisible();

  await expect(page.locator(".resource-row")).toHaveCount(2);

  await page.locator("#resource-search").fill("roadmap");

  await expect(page.locator(".resource-row")).toHaveCount(1);
  await expect(page.locator(".resource-row h3", { hasText: "Roadmap doc" })).toBeVisible();
  await expect(page.getByText(/Visible 1 of 2 resources/i)).toBeVisible();
});

test("shows settings runtime counts and sync status", async ({ page }) => {
  await installApiMocks(page);

  await page.goto("/");

  await page.getByRole("button", { name: "Settings" }).click();
  await expect(page.getByRole("heading", { name: "Runtime Settings" })).toBeVisible();

  const syncCard = page.locator("article.settings-card", { hasText: "Sync Runtime" });
  await expect(syncCard).toContainText("connected");

  const loadedCard = page.locator("article.settings-card", { hasText: "Loaded Records" });
  await expect(loadedCard.locator(".settings-row", { hasText: "Resources" })).toContainText("2");
  await expect(loadedCard.locator(".settings-row", { hasText: "Todos" })).toContainText("1");
  await expect(loadedCard.locator(".settings-row", { hasText: "Reminders" })).toContainText("1");
});

test("filters resources with graph controls and resets filters", async ({ page }) => {
  await installApiMocks(page);

  await page.goto("/");

  const graphControls = page.locator("section.graph-controls");
  const resourceRows = page.locator(".resource-row");

  await expect(resourceRows).toHaveCount(2);

  await page.locator("#resource-search").fill("notes");
  await expect(resourceRows).toHaveCount(1);
  await expect(page.locator(".resource-row h3", { hasText: "Research notes" })).toBeVisible();

  await graphControls.locator("select").first().selectOption("research");
  await expect(resourceRows).toHaveCount(1);
  await expect(page.locator(".resource-row h3", { hasText: "Research notes" })).toBeVisible();

  await graphControls.getByLabel("User overrides only").check();
  await expect(resourceRows).toHaveCount(1);
  await expect(page.locator(".resource-row h3", { hasText: "Research notes" })).toBeVisible();

  await graphControls.getByRole("button", { name: "Reset filters" }).click();
  await expect(page.locator("#resource-search")).toHaveValue("");
  await expect(graphControls.locator("select").first()).toHaveValue("all");
  await graphControls.getByLabel("User overrides only").uncheck();
  await expect(resourceRows).toHaveCount(2);
});

test("toggles graph view mode buttons", async ({ page }) => {
  await installApiMocks(page);

  await page.goto("/");

  const graphControls = page.locator("section.graph-controls");
  const viewModeToggle = graphControls.locator(".mode-toggle");
  const mode2d = viewModeToggle.getByRole("button", { name: "2D" });
  const mode3d = viewModeToggle.getByRole("button", { name: "3D" });

  await expect(mode3d).toHaveClass(/is-active/);
  await expect(mode2d).not.toHaveClass(/is-active/);

  await mode2d.click({ force: true });
  await expect(mode2d).toHaveClass(/is-active/);
  await expect(mode3d).not.toHaveClass(/is-active/);

  await mode3d.click({ force: true });
  await expect(mode3d).toHaveClass(/is-active/);
  await expect(mode2d).not.toHaveClass(/is-active/);
});

test("updates override-tagged count when filtering overrides", async ({ page }) => {
  await installApiMocks(page);

  await page.goto("/");

  const resourceList = page.locator("section.resource-list");
  const overrideToggle = page.getByLabel("User overrides only");

  await expect(resourceList.getByText("2 visible, 1 override-tagged")).toBeVisible();

  await overrideToggle.check();

  await expect(resourceList.getByText("1 visible, 1 override-tagged")).toBeVisible();
  await expect(page.locator(".resource-row")).toHaveCount(1);
});

test("updates graph meta counts and empty state with filters", async ({ page }) => {
  await installApiMocks(page);

  await page.goto("/");

  const graphControls = page.locator("section.graph-controls");
  const graphMeta = page.locator("section.graph-canvas .graph-meta");

  await expect(graphMeta.getByText("2 resource nodes")).toBeVisible();
  await expect(graphMeta.getByText("2 category hubs")).toBeVisible();
  await expect(graphMeta.getByText("2 links")).toBeVisible();

  await graphControls.getByLabel("User overrides only").check();

  await expect(graphMeta.getByText("1 resource nodes")).toBeVisible();
  await expect(graphMeta.getByText("1 category hubs")).toBeVisible();
  await expect(graphMeta.getByText("1 links")).toBeVisible();

  await page.locator("#resource-search").fill("no-match");
  await expect(page.locator(".graph-empty")).toBeVisible();

  await page.locator("#resource-search").fill("");
  await graphControls.getByLabel("User overrides only").uncheck();
  await expect(graphMeta.getByText("2 resource nodes")).toBeVisible();
});

test("selects a resource via graph test hook", async ({ page }) => {
  await installApiMocks(page);

  await page.goto("/");

  const form = page.locator("section.resource-form");
  await expect(page.locator(".resource-row")).toHaveCount(2);
  await expect(form.getByRole("heading", { name: "Add Resource" })).toBeVisible();

  await page.waitForFunction(() => typeof (globalThis as any).__graphTest?.selectResource === "function");

  await page.evaluate(() => {
    (globalThis as any).__graphTest?.selectResource("res-1");
  });

  await expect(form.getByRole("heading", { name: "Edit Resource" })).toBeVisible();
  await expect(form.getByText("Selected: Roadmap doc")).toBeVisible();
  await expect(page.locator(".resource-row.is-selected")).toContainText("Roadmap doc");
});

test("collapses and expands the sidebar", async ({ page }) => {
  await installApiMocks(page);

  await page.goto("/");

  const sidebar = page.locator("aside.sidebar");

  await expect(sidebar).not.toHaveClass(/is-collapsed/);
  await expect(sidebar.locator(".brand-title")).toHaveCount(1);

  await sidebar.getByRole("button", { name: "Collapse" }).click();
  await expect(sidebar).toHaveClass(/is-collapsed/);
  await expect(sidebar.locator(".brand-title")).toHaveCount(0);

  await sidebar.getByRole("button", { name: "Expand" }).click();
  await expect(sidebar).not.toHaveClass(/is-collapsed/);
  await expect(sidebar.locator(".brand-title")).toHaveCount(1);
});
