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

type ChatFailureMode = "none" | "backend" | "network" | "malformed";

interface ApiMockState {
  resources: ResourceItem[];
  todos: TodoItem[];
  reminders: ReminderItem[];
  nextResourceID: number;
  chatFailureMode: ChatFailureMode;
  nextChatAction: string;
  nextChatMessage: string;
  mutateResourcesOnChat: boolean;
  lastChatMessage: string | null;
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

function setChatFailure(state: ApiMockState, mode: ChatFailureMode) {
  state.chatFailureMode = mode;
}

function setChatResult(state: ApiMockState, action: string, message: string, mutateResourcesOnChat: boolean) {
  state.nextChatAction = action;
  state.nextChatMessage = message;
  state.mutateResourcesOnChat = mutateResourcesOnChat;
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
    ],
    todos: [],
    reminders: [],
    nextResourceID: 2,
    chatFailureMode: "none",
    nextChatAction: "list_resources",
    nextChatMessage: "Found resources.",
    mutateResourcesOnChat: false,
    lastChatMessage: null,
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

    if (pathname === "/api/v1/chat/commands" && method === "POST") {
      const payload = (JSON.parse(request.postData() ?? "{}") ?? {}) as Record<string, unknown>;
      state.lastChatMessage = typeof payload.message === "string" ? payload.message : "";

      if (state.chatFailureMode === "backend") {
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

      if (state.chatFailureMode === "network") {
        await route.abort("failed");
        return;
      }

      if (state.chatFailureMode === "malformed") {
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

      if (state.mutateResourcesOnChat) {
        const urlValue = "https://example.com/chat-resource";
        const now = new Date().toISOString();
        const nextID = `res-${state.nextResourceID}`;
        const created: ResourceItem = {
          id: nextID,
          url: urlValue,
          host: safeHost(urlValue),
          title: "Chat-created resource",
          summary: "Created from chat action.",
          category_id: `cat-${state.nextResourceID}`,
          category_name: "ChatOps",
          user_override: false,
          created_at: now,
          updated_at: now,
        };
        state.nextResourceID += 1;
        state.resources = [created, ...state.resources];
      }

      await fulfillEnvelope(route, { action: state.nextChatAction, message: state.nextChatMessage });
      return;
    }

    await fulfillEnvelope(route, []);
  });

  return state;
}

test("sends chat command and appends assistant reply", async ({ page }) => {
  const state = await installApiMocks(page);
  setChatResult(state, "list_resources", "Found 1 resource.", false);

  await page.goto("/");

  await page.getByRole("button", { name: "Chat" }).click();
  await expect(page.getByRole("heading", { name: "Chat Layout" })).toBeVisible();

  const dock = page.locator("section.chat-dock");
  const chatLog = dock.locator(".chat-log");

  await dock.getByPlaceholder("Type a command...").fill("list resources");
  await dock.getByRole("button", { name: "Send" }).click();

  await expect.poll(() => state.lastChatMessage).toBe("list resources");
  await expect(chatLog.locator(".chat-msg.is-user").getByText("list resources")).toBeVisible();
  await expect(chatLog.getByText("Found 1 resource.")).toBeVisible();
  await expect(dock.getByPlaceholder("Type a command...")).toHaveValue("");
});

test("shows chat error message when command fails", async ({ page }) => {
  const state = await installApiMocks(page);
  setChatFailure(state, "backend");

  await page.goto("/");

  await page.getByRole("button", { name: "Chat" }).click();
  await expect(page.getByRole("heading", { name: "Chat Layout" })).toBeVisible();

  const dock = page.locator("section.chat-dock");
  const chatLog = dock.locator(".chat-log");

  await dock.getByPlaceholder("Type a command...").fill("create category research");
  await dock.getByRole("button", { name: "Send" }).click();

  await expect.poll(() => state.lastChatMessage).toBe("create category research");
  await expect(chatLog.getByText(/Command failed: Chat command rejected from mock/i)).toBeVisible();
});

test("reloads resources after chat mutation action", async ({ page }) => {
  const state = await installApiMocks(page);
  setChatResult(state, "resource_created", "Resource created.", true);

  await page.goto("/");

  await page.getByRole("button", { name: "Chat" }).click();
  await expect(page.getByRole("heading", { name: "Chat Layout" })).toBeVisible();

  const dock = page.locator("section.chat-dock");
  const chatLog = dock.locator(".chat-log");

  await dock.getByPlaceholder("Type a command...").fill("create resource https://example.com/chat-resource");
  await dock.getByRole("button", { name: "Send" }).click();

  await expect.poll(() => state.resources.length).toBe(2);
  await expect(chatLog.getByText("Resource created.")).toBeVisible();
  await expect(page.locator(".resource-row h3", { hasText: "Chat-created resource" })).toBeVisible();
});
