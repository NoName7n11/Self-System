import { expect, test, type Locator, type Page, type Route } from "@playwright/test";

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

interface ApiMockState {
  resources: ResourceItem[];
  nextResourceID: number;
  failCreate: boolean;
  abortCreateNetwork: boolean;
  malformedCreate: boolean;
  failUpdateIDs: Set<string>;
  timeoutUpdateIDs: Set<string>;
  malformedUpdateIDs: Set<string>;
  failDeleteIDs: Set<string>;
  timeoutDeleteIDs: Set<string>;
  malformedDeleteSuccessIDs: Set<string>;
  emptyDeleteSuccessIDs: Set<string>;
  lastCreatePayload: Record<string, unknown> | null;
  lastUpdatePayload: Record<string, unknown> | null;
  lastDeleteID: string | null;
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
        // no-op: these tests do not rely on outbound websocket traffic.
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
type DeleteFailureMode = "backend" | "timeout";
type DeleteSuccessBodyMode = "malformed" | "empty";

function setCreateFailure(state: ApiMockState, mode: CreateFailureMode) {
  state.failCreate = mode === "backend";
  state.abortCreateNetwork = mode === "network";
  state.malformedCreate = mode === "malformed";
}

function setUpdateFailure(state: ApiMockState, resourceID: string, mode: UpdateFailureMode) {
  state.failUpdateIDs.delete(resourceID);
  state.timeoutUpdateIDs.delete(resourceID);
  state.malformedUpdateIDs.delete(resourceID);

  if (mode === "backend") {
    state.failUpdateIDs.add(resourceID);
    return;
  }

  if (mode === "timeout") {
    state.timeoutUpdateIDs.add(resourceID);
    return;
  }

  state.malformedUpdateIDs.add(resourceID);
}

function setDeleteFailure(state: ApiMockState, resourceID: string, mode: DeleteFailureMode) {
  state.failDeleteIDs.delete(resourceID);
  state.timeoutDeleteIDs.delete(resourceID);

  if (mode === "timeout") {
    state.timeoutDeleteIDs.add(resourceID);
    return;
  }

  state.failDeleteIDs.add(resourceID);
}

function setDeleteSuccessBody(state: ApiMockState, resourceID: string, mode: DeleteSuccessBodyMode) {
  state.malformedDeleteSuccessIDs.delete(resourceID);
  state.emptyDeleteSuccessIDs.delete(resourceID);

  if (mode === "malformed") {
    state.malformedDeleteSuccessIDs.add(resourceID);
    return;
  }

  state.emptyDeleteSuccessIDs.add(resourceID);
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

async function selectRoadmapResource(page: Page, form: Locator) {
  const row = page.locator(".resource-row", { hasText: "Roadmap doc" }).first();
  const editHeading = form.getByRole("heading", { name: "Edit Resource" });

  await expect(row).toBeVisible();

  for (let attempt = 0; attempt < 3; attempt += 1) {
    await row.dispatchEvent("click");
    if (await editHeading.isVisible()) {
      return;
    }

    await row.click({ force: true });
    if (await editHeading.isVisible()) {
      return;
    }
  }

  await expect(editHeading).toBeVisible();
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
    nextResourceID: 2,
    failCreate: false,
    abortCreateNetwork: false,
    malformedCreate: false,
    failUpdateIDs: new Set<string>(),
    timeoutUpdateIDs: new Set<string>(),
    malformedUpdateIDs: new Set<string>(),
    failDeleteIDs: new Set<string>(),
    timeoutDeleteIDs: new Set<string>(),
    malformedDeleteSuccessIDs: new Set<string>(),
    emptyDeleteSuccessIDs: new Set<string>(),
    lastCreatePayload: null,
    lastUpdatePayload: null,
    lastDeleteID: null,
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
      const payload = (JSON.parse(request.postData() ?? "{}") ?? {}) as Record<string, unknown>;
      state.lastCreatePayload = payload;

      if (state.failCreate) {
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

      if (state.abortCreateNetwork) {
        await route.abort("failed");
        return;
      }

      if (state.malformedCreate) {
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

      const urlValue = typeof payload.url === "string" ? payload.url : "";
      const now = new Date().toISOString();
      const nextID = `res-${state.nextResourceID}`;
      const created: ResourceItem = {
        id: nextID,
        url: urlValue,
        host: safeHost(urlValue),
        title: typeof payload.title === "string" ? payload.title : "",
        summary: typeof payload.summary === "string" ? payload.summary : "",
        category_id: `cat-${state.nextResourceID}`,
        category_name: typeof payload.category_name === "string" ? payload.category_name : "Unsorted",
        user_override: false,
        created_at: now,
        updated_at: now,
      };

      state.nextResourceID += 1;
      state.resources = [created, ...state.resources];

      await fulfillEnvelope(route, created, 201);
      return;
    }

    if (pathname.startsWith("/api/v1/resources/") && method === "PUT") {
      const payload = (JSON.parse(request.postData() ?? "{}") ?? {}) as Record<string, unknown>;
      state.lastUpdatePayload = payload;

      const resourceID = pathname.split("/").pop() ?? "";
      const target = state.resources.find((item) => item.id === resourceID);

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

      if (state.failUpdateIDs.has(resourceID)) {
        await route.fulfill({
          status: 500,
          headers: {
            "content-type": "application/json",
            ...corsHeaders,
          },
          body: JSON.stringify({ error: "Failed to update resource from mock", code: "mock_resource_update_error" }),
        });
        return;
      }

      if (state.timeoutUpdateIDs.has(resourceID)) {
        await route.fulfill({
          status: 504,
          headers: {
            "content-type": "application/json",
            ...corsHeaders,
          },
          body: JSON.stringify({ error: "Resource update timed out from mock", code: "mock_resource_timeout" }),
        });
        return;
      }

      if (state.malformedUpdateIDs.has(resourceID)) {
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

      const urlValue = typeof payload.url === "string" ? payload.url : target.url;
      const updated: ResourceItem = {
        ...target,
        url: urlValue,
        host: safeHost(urlValue) || target.host,
        title: typeof payload.title === "string" ? payload.title : target.title,
        summary: typeof payload.summary === "string" ? payload.summary : target.summary,
        category_name: typeof payload.category_name === "string" ? payload.category_name : target.category_name,
        updated_at: new Date().toISOString(),
      };

      state.resources = state.resources.map((item) => (item.id === resourceID ? updated : item));

      await fulfillEnvelope(route, updated);
      return;
    }

    if (pathname.startsWith("/api/v1/resources/") && method === "DELETE") {
      const resourceID = pathname.split("/").pop() ?? "";
      state.lastDeleteID = resourceID;

      if (state.failDeleteIDs.has(resourceID)) {
        await route.fulfill({
          status: 500,
          headers: {
            "content-type": "application/json",
            ...corsHeaders,
          },
          body: JSON.stringify({ error: "Failed to delete resource from mock", code: "mock_resource_delete_error" }),
        });
        return;
      }

      if (state.timeoutDeleteIDs.has(resourceID)) {
        await route.fulfill({
          status: 504,
          headers: {
            "content-type": "application/json",
            ...corsHeaders,
          },
          body: JSON.stringify({ error: "Resource delete timed out from mock", code: "mock_resource_delete_timeout" }),
        });
        return;
      }

      if (state.malformedDeleteSuccessIDs.has(resourceID)) {
        state.resources = state.resources.filter((item) => item.id !== resourceID);
        await route.fulfill({
          status: 200,
          headers: {
            "content-type": "application/json",
            ...corsHeaders,
          },
          body: "{not-json",
        });
        return;
      }

      if (state.emptyDeleteSuccessIDs.has(resourceID)) {
        state.resources = state.resources.filter((item) => item.id !== resourceID);
        await route.fulfill({
          status: 204,
          headers: {
            ...corsHeaders,
          },
        });
        return;
      }

      state.resources = state.resources.filter((item) => item.id !== resourceID);
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

    if (pathname === "/api/v1/todos" && method === "GET") {
      await fulfillEnvelope(route, []);
      return;
    }

    if (pathname === "/api/v1/reminders" && method === "GET") {
      await fulfillEnvelope(route, []);
      return;
    }

    await fulfillEnvelope(route, []);
  });

  return state;
}

test("creates a resource and shows it in the ledger", async ({ page }) => {
  const state = await installApiMocks(page);

  await page.goto("/");
  await expect(page.getByRole("heading", { name: "Resource Ledger" })).toBeVisible();

  const form = page.locator("section.resource-form");

  await form.getByLabel("URL").fill("https://example.com/new-resource");
  await form.getByLabel("Title").fill("New Resource");
  await form.getByLabel("Category").fill("Research");
  await form.getByLabel("Summary").fill("Resource create flow coverage");
  await form.getByRole("button", { name: "Add As New" }).click();

  await expect.poll(() => state.lastCreatePayload?.category_name).toBe("Research");
  await expect.poll(() => state.resources.length).toBe(2);
  await expect(page.locator(".resource-row h3", { hasText: "New Resource" }).first()).toBeVisible();
  await expect(form.getByRole("heading", { name: "Edit Resource" })).toBeVisible();
});

test("updates selected resource and sends category_name", async ({ page }) => {
  const state = await installApiMocks(page);

  await page.goto("/");

  const form = page.locator("section.resource-form");

  await selectRoadmapResource(page, form);
  await expect(form.getByRole("heading", { name: "Edit Resource" })).toBeVisible();

  await form.getByLabel("Title").fill("Roadmap doc updated");
  await form.getByLabel("Category").fill("Deep Work");
  await form.getByRole("button", { name: "Update Selected" }).click();

  await expect.poll(() => state.lastUpdatePayload?.category_name).toBe("Deep Work");
  await expect.poll(() => Object.prototype.hasOwnProperty.call(state.lastUpdatePayload ?? {}, "category")).toBe(false);
  await expect(page.locator(".resource-row h3", { hasText: "Roadmap doc updated" }).first()).toBeVisible();
  await expect(page.locator(".resource-row .resource-chip").first()).toHaveText("Deep Work");
});

test("shows create error and preserves existing resources", async ({ page }) => {
  const state = await installApiMocks(page);
  setCreateFailure(state, "backend");

  await page.goto("/");

  const form = page.locator("section.resource-form");

  await form.getByLabel("URL").fill("https://example.com/should-fail");
  await form.getByLabel("Title").fill("Create should fail");
  await form.getByLabel("Category").fill("Research");
  await form.getByRole("button", { name: "Add As New" }).click();

  await expect.poll(() => state.lastCreatePayload?.title).toBe("Create should fail");
  await expect(page.getByText("Failed to create resource from mock")).toBeVisible();
  await expect.poll(() => state.resources.length).toBe(1);
  await expect(page.getByText("Create should fail")).toHaveCount(0);
});

test("shows create network error and preserves existing resources", async ({ page }) => {
  const state = await installApiMocks(page);
  setCreateFailure(state, "network");

  await page.goto("/");

  const form = page.locator("section.resource-form");

  await form.getByLabel("URL").fill("https://example.com/network-fail");
  await form.getByLabel("Title").fill("Create network should fail");
  await form.getByLabel("Category").fill("Research");
  await form.getByRole("button", { name: "Add As New" }).click();

  await expect.poll(() => state.lastCreatePayload?.title).toBe("Create network should fail");
  await expect(page.getByText(/Failed to fetch|NetworkError|Load failed/i)).toBeVisible();
  await expect.poll(() => state.resources.length).toBe(1);
  await expect(page.getByText("Create network should fail")).toHaveCount(0);
});

test("shows create envelope error and preserves existing resources", async ({ page }) => {
  const state = await installApiMocks(page);
  setCreateFailure(state, "malformed");

  await page.goto("/");

  const form = page.locator("section.resource-form");

  await form.getByLabel("URL").fill("https://example.com/malformed-fail");
  await form.getByLabel("Title").fill("Create malformed should fail");
  await form.getByLabel("Category").fill("Research");
  await form.getByRole("button", { name: "Add As New" }).click();

  await expect.poll(() => state.lastCreatePayload?.title).toBe("Create malformed should fail");
  await expect(page.getByText("API response did not include data")).toBeVisible();
  await expect.poll(() => state.resources.length).toBe(1);
  await expect(page.getByText("Create malformed should fail")).toHaveCount(0);
});

test("shows update error and keeps previous resource state", async ({ page }) => {
  const state = await installApiMocks(page);

  await page.goto("/");

  const form = page.locator("section.resource-form");

  await form.getByLabel("URL").fill("https://example.com/update-backend-target");
  await form.getByLabel("Title").fill("Update backend target");
  await form.getByLabel("Category").fill("Research");
  await form.getByRole("button", { name: "Add As New" }).click();

  await expect.poll(() => state.resources.length).toBe(2);
  await expect.poll(() => state.resources[0]?.id).toBe("res-2");
  await expect(form.getByRole("heading", { name: "Edit Resource" })).toBeVisible();
  await expect(form.getByRole("button", { name: "Update Selected" })).toBeEnabled();
  setUpdateFailure(state, "res-2", "backend");

  await form.getByLabel("Title").fill("Should backend fail");
  await form.getByRole("button", { name: "Update Selected" }).click();

  await expect.poll(() => state.lastUpdatePayload?.title).toBe("Should backend fail");
  await expect(page.getByText("Failed to update resource from mock")).toBeVisible();
  await expect.poll(() => state.resources.find((item) => item.id === "res-2")?.title).toBe("Update backend target");
  await expect(page.locator(".resource-row h3", { hasText: "Update backend target" }).first()).toBeVisible();
});

test("shows update envelope error and keeps previous resource state", async ({ page }) => {
  const state = await installApiMocks(page);

  await page.goto("/");

  const form = page.locator("section.resource-form");

  await selectRoadmapResource(page, form);
  await expect(form.getByRole("heading", { name: "Edit Resource" })).toBeVisible();
  await expect(form.getByRole("button", { name: "Update Selected" })).toBeEnabled();
  setUpdateFailure(state, "res-1", "malformed");

  await form.getByLabel("Title").fill("Should not persist");
  await form.getByRole("button", { name: "Update Selected" }).click();

  await expect.poll(() => state.lastUpdatePayload?.title).toBe("Should not persist");
  await expect(page.getByText("API response did not include data")).toBeVisible();
  await expect.poll(() => state.resources.find((item) => item.id === "res-1")?.title).toBe("Roadmap doc");
  await expect(page.locator(".resource-row h3", { hasText: "Roadmap doc" }).first()).toBeVisible();
});

test("shows update timeout error and keeps previous resource state", async ({ page }) => {
  const state = await installApiMocks(page);

  await page.goto("/");

  const form = page.locator("section.resource-form");

  await form.getByLabel("URL").fill("https://example.com/update-timeout-target");
  await form.getByLabel("Title").fill("Update timeout target");
  await form.getByLabel("Category").fill("Research");
  await form.getByRole("button", { name: "Add As New" }).click();

  await expect.poll(() => state.resources.length).toBe(2);
  await expect.poll(() => state.resources[0]?.id).toBe("res-2");
  await expect(form.getByRole("heading", { name: "Edit Resource" })).toBeVisible();
  await expect(form.getByRole("button", { name: "Update Selected" })).toBeEnabled();
  setUpdateFailure(state, "res-2", "timeout");

  await form.getByLabel("Title").fill("Should timeout");
  await form.getByRole("button", { name: "Update Selected" }).click();

  await expect.poll(() => state.lastUpdatePayload?.title).toBe("Should timeout");
  await expect(page.getByText("Resource update timed out from mock")).toBeVisible();
  await expect.poll(() => state.resources.find((item) => item.id === "res-2")?.title).toBe("Update timeout target");
  await expect(page.locator(".resource-row h3", { hasText: "Update timeout target" }).first()).toBeVisible();
});

test("deletes selected resource from the ledger", async ({ page }) => {
  const state = await installApiMocks(page);

  await page.goto("/");

  const form = page.locator("section.resource-form");

  await selectRoadmapResource(page, form);
  await expect(form.getByRole("heading", { name: "Edit Resource" })).toBeVisible();
  await expect(form.getByRole("button", { name: "Delete Selected" })).toBeEnabled();
  await form.getByRole("button", { name: "Delete Selected" }).click();

  await expect.poll(() => state.lastDeleteID).toBe("res-1");
  await expect.poll(() => state.resources.length).toBe(0);
  await expect(page.getByText("No resources match the current filters.")).toBeVisible();
  await expect(form.getByRole("heading", { name: "Add Resource" })).toBeVisible();
});

test("shows delete error and keeps selected resource", async ({ page }) => {
  const state = await installApiMocks(page);
  setDeleteFailure(state, "res-1", "backend");

  await page.goto("/");

  const form = page.locator("section.resource-form");

  await selectRoadmapResource(page, form);
  await expect(form.getByRole("heading", { name: "Edit Resource" })).toBeVisible();
  await expect(form.getByRole("button", { name: "Delete Selected" })).toBeEnabled();
  await form.getByRole("button", { name: "Delete Selected" }).click();

  await expect.poll(() => state.lastDeleteID).toBe("res-1");
  await expect(page.getByText("Failed to delete resource from mock")).toBeVisible();
  await expect.poll(() => state.resources.length).toBe(1);
  await expect(page.locator(".resource-row h3", { hasText: "Roadmap doc" }).first()).toBeVisible();
});

test("shows delete timeout error and keeps selected resource", async ({ page }) => {
  const state = await installApiMocks(page);
  setDeleteFailure(state, "res-1", "timeout");

  await page.goto("/");

  const form = page.locator("section.resource-form");

  await selectRoadmapResource(page, form);
  await expect(form.getByRole("heading", { name: "Edit Resource" })).toBeVisible();
  await expect(form.getByRole("button", { name: "Delete Selected" })).toBeEnabled();
  await form.getByRole("button", { name: "Delete Selected" }).click();

  await expect.poll(() => state.lastDeleteID).toBe("res-1");
  await expect(page.getByText("Resource delete timed out from mock")).toBeVisible();
  await expect.poll(() => state.resources.length).toBe(1);
  await expect(page.locator(".resource-row h3", { hasText: "Roadmap doc" }).first()).toBeVisible();
});

test("deletes selected resource when delete success body is malformed", async ({ page }) => {
  const state = await installApiMocks(page);
  setDeleteSuccessBody(state, "res-1", "malformed");

  await page.goto("/");

  const form = page.locator("section.resource-form");

  await selectRoadmapResource(page, form);
  await expect(form.getByRole("heading", { name: "Edit Resource" })).toBeVisible();
  await expect(form.getByRole("button", { name: "Delete Selected" })).toBeEnabled();
  await form.getByRole("button", { name: "Delete Selected" }).click();

  await expect.poll(() => state.lastDeleteID).toBe("res-1");
  await expect.poll(() => state.resources.length).toBe(0);
  await expect(page.getByText("No resources match the current filters.")).toBeVisible();
  await expect(form.getByRole("heading", { name: "Add Resource" })).toBeVisible();
});

test("deletes selected resource when delete success body is empty", async ({ page }) => {
  const state = await installApiMocks(page);
  setDeleteSuccessBody(state, "res-1", "empty");

  await page.goto("/");

  const form = page.locator("section.resource-form");

  await selectRoadmapResource(page, form);
  await expect(form.getByRole("heading", { name: "Edit Resource" })).toBeVisible();
  await expect(form.getByRole("button", { name: "Delete Selected" })).toBeEnabled();
  await form.getByRole("button", { name: "Delete Selected" }).click();

  await expect.poll(() => state.lastDeleteID).toBe("res-1");
  await expect.poll(() => state.resources.length).toBe(0);
  await expect(page.getByText("No resources match the current filters.")).toBeVisible();
  await expect(form.getByRole("heading", { name: "Add Resource" })).toBeVisible();
});
