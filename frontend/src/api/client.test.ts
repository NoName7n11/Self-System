import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  createReminder,
  createTodo,
  createResource,
  deleteReminder,
  deleteResource,
  deleteTodo,
  listResources,
  listReminders,
  listTodos,
  sendChatCommand,
  updateReminder,
  updateResource,
  updateTodo,
} from "./client";

describe("api client envelope contracts", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.clearAllMocks();
    vi.unstubAllGlobals();
  });

  it("throws when todo update success response omits data envelope", async () => {
    const fetchMock = vi.mocked(globalThis.fetch);
    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify({ message: "ok" }), {
        status: 200,
        headers: { "content-type": "application/json" },
      }),
    );

    await expect(updateTodo("todo-1", { title: "Updated" })).rejects.toThrow("API response did not include data");
  });

  it("throws when reminder update success response has empty body", async () => {
    const fetchMock = vi.mocked(globalThis.fetch);
    fetchMock.mockResolvedValueOnce(
      new Response("", {
        status: 200,
        headers: { "content-type": "application/json" },
      }),
    );

    await expect(updateReminder("rem-1", { message: "Updated" })).rejects.toThrow("API response did not include data");
  });

  it("throws when todo create success response omits data envelope", async () => {
    const fetchMock = vi.mocked(globalThis.fetch);
    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify({ message: "ok" }), {
        status: 200,
        headers: { "content-type": "application/json" },
      }),
    );

    await expect(createTodo({ title: "Todo create" })).rejects.toThrow("API response did not include data");
  });

  it("throws when reminder create success response body is malformed", async () => {
    const fetchMock = vi.mocked(globalThis.fetch);
    fetchMock.mockResolvedValueOnce(
      new Response("{not-json", {
        status: 200,
        headers: { "content-type": "application/json" },
      }),
    );

    await expect(createReminder({ title: "Reminder create" })).rejects.toThrow("API response did not include data");
  });

  it("throws when todo list success response omits data envelope", async () => {
    const fetchMock = vi.mocked(globalThis.fetch);
    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify({ message: "ok" }), {
        status: 200,
        headers: { "content-type": "application/json" },
      }),
    );

    await expect(listTodos()).rejects.toThrow("API response did not include data");
  });

  it("throws when reminder list success response body is malformed", async () => {
    const fetchMock = vi.mocked(globalThis.fetch);
    fetchMock.mockResolvedValueOnce(
      new Response("{not-json", {
        status: 200,
        headers: { "content-type": "application/json" },
      }),
    );

    await expect(listReminders()).rejects.toThrow("API response did not include data");
  });

  it("returns empty array when todo list data envelope is non-array", async () => {
    const fetchMock = vi.mocked(globalThis.fetch);
    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify({ data: { unexpected: true } }), {
        status: 200,
        headers: { "content-type": "application/json" },
      }),
    );

    await expect(listTodos()).resolves.toEqual([]);
  });

  it("throws when resource list success response omits data envelope", async () => {
    const fetchMock = vi.mocked(globalThis.fetch);
    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify({ message: "ok" }), {
        status: 200,
        headers: { "content-type": "application/json" },
      }),
    );

    await expect(listResources()).rejects.toThrow("API response did not include data");
  });

  it("throws when resource list success response body is malformed", async () => {
    const fetchMock = vi.mocked(globalThis.fetch);
    fetchMock.mockResolvedValueOnce(
      new Response("{not-json", {
        status: 200,
        headers: { "content-type": "application/json" },
      }),
    );

    await expect(listResources()).rejects.toThrow("API response did not include data");
  });

  it("returns empty array when resource list data envelope is non-array", async () => {
    const fetchMock = vi.mocked(globalThis.fetch);
    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify({ data: { unexpected: true } }), {
        status: 200,
        headers: { "content-type": "application/json" },
      }),
    );

    await expect(listResources()).resolves.toEqual([]);
  });

  it("throws when resource create success response omits data envelope", async () => {
    const fetchMock = vi.mocked(globalThis.fetch);
    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify({ message: "ok" }), {
        status: 200,
        headers: { "content-type": "application/json" },
      }),
    );

    await expect(createResource({ url: "https://example.com/new" })).rejects.toThrow("API response did not include data");
  });

  it("throws when resource update success response body is malformed", async () => {
    const fetchMock = vi.mocked(globalThis.fetch);
    fetchMock.mockResolvedValueOnce(
      new Response("{not-json", {
        status: 200,
        headers: { "content-type": "application/json" },
      }),
    );

    await expect(updateResource("res-1", { title: "Updated" })).rejects.toThrow("API response did not include data");
  });

  it("uses category_name key for resource update requests", async () => {
    const fetchMock = vi.mocked(globalThis.fetch);
    fetchMock.mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          data: {
            id: "res-1",
            url: "https://example.com",
            host: "example.com",
            title: "Updated",
            summary: "Summary",
            category_id: "cat-1",
            category_name: "Research",
            user_override: false,
            created_at: "2026-04-21T10:00:00Z",
            updated_at: "2026-04-21T10:05:00Z",
          },
        }),
        {
          status: 200,
          headers: { "content-type": "application/json" },
        },
      ),
    );

    await updateResource("res-1", { categoryName: "Research", title: "Updated" });

    const call = fetchMock.mock.calls[0];
    expect(call).toBeDefined();

    const requestOptions = call?.[1] as RequestInit;
    const requestBody = JSON.parse(String(requestOptions.body)) as Record<string, unknown>;

    expect(requestBody.category_name).toBe("Research");
    expect(requestBody.category).toBeUndefined();
  });

  it("accepts malformed success body for todo delete no-data path", async () => {
    const fetchMock = vi.mocked(globalThis.fetch);
    fetchMock.mockResolvedValueOnce(
      new Response("{not-json", {
        status: 200,
        headers: { "content-type": "application/json" },
      }),
    );

    await expect(deleteTodo("todo-1")).resolves.toBeUndefined();
  });

  it("accepts empty success body for resource delete no-data path", async () => {
    const fetchMock = vi.mocked(globalThis.fetch);
    fetchMock.mockResolvedValueOnce(
      new Response(null, {
        status: 204,
      }),
    );

    await expect(deleteResource("res-1")).resolves.toBeUndefined();
  });

  it("accepts empty success body for reminder delete no-data path", async () => {
    const fetchMock = vi.mocked(globalThis.fetch);
    fetchMock.mockResolvedValueOnce(
      new Response(null, {
        status: 204,
      }),
    );

    await expect(deleteReminder("rem-1")).resolves.toBeUndefined();
  });

  it("surfaces envelope message for todo delete non-ok response", async () => {
    const fetchMock = vi.mocked(globalThis.fetch);
    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify({ message: "delete failed by contract" }), {
        status: 500,
        headers: { "content-type": "application/json" },
      }),
    );

    await expect(deleteTodo("todo-1")).rejects.toThrow("delete failed by contract");
  });

  it("surfaces envelope error for todo create non-ok response", async () => {
    const fetchMock = vi.mocked(globalThis.fetch);
    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify({ error: "todo create denied" }), {
        status: 400,
        headers: { "content-type": "application/json" },
      }),
    );

    await expect(createTodo({ title: "Todo create" })).rejects.toThrow("todo create denied");
  });

  it("surfaces envelope message for reminder list non-ok response", async () => {
    const fetchMock = vi.mocked(globalThis.fetch);
    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify({ message: "reminder list denied" }), {
        status: 503,
        headers: { "content-type": "application/json" },
      }),
    );

    await expect(listReminders()).rejects.toThrow("reminder list denied");
  });

  it("surfaces envelope error for resource list non-ok response", async () => {
    const fetchMock = vi.mocked(globalThis.fetch);
    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify({ error: "resource list denied" }), {
        status: 500,
        headers: { "content-type": "application/json" },
      }),
    );

    await expect(listResources()).rejects.toThrow("resource list denied");
  });

  it("falls back to status text when todo list non-ok body is malformed", async () => {
    const fetchMock = vi.mocked(globalThis.fetch);
    fetchMock.mockResolvedValueOnce(
      new Response("<html>gateway timeout</html>", {
        status: 502,
        headers: { "content-type": "text/html" },
      }),
    );

    await expect(listTodos()).rejects.toThrow("Request failed (502)");
  });

  it("falls back to status text when resource list non-ok body is malformed", async () => {
    const fetchMock = vi.mocked(globalThis.fetch);
    fetchMock.mockResolvedValueOnce(
      new Response("<html>gateway timeout</html>", {
        status: 502,
        headers: { "content-type": "text/html" },
      }),
    );

    await expect(listResources()).rejects.toThrow("Request failed (502)");
  });

  it("falls back to status text when reminder create non-ok body is malformed", async () => {
    const fetchMock = vi.mocked(globalThis.fetch);
    fetchMock.mockResolvedValueOnce(
      new Response("<html>gateway timeout</html>", {
        status: 502,
        headers: { "content-type": "text/html" },
      }),
    );

    await expect(createReminder({ title: "Reminder create" })).rejects.toThrow("Request failed (502)");
  });

  it("falls back to status text when reminder delete non-ok body is malformed", async () => {
    const fetchMock = vi.mocked(globalThis.fetch);
    fetchMock.mockResolvedValueOnce(
      new Response("<html>gateway timeout</html>", {
        status: 504,
        headers: { "content-type": "text/html" },
      }),
    );

    await expect(deleteReminder("rem-1")).rejects.toThrow("Request failed (504)");
  });

  it("surfaces envelope error for resource delete non-ok response", async () => {
    const fetchMock = vi.mocked(globalThis.fetch);
    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify({ error: "resource delete denied" }), {
        status: 403,
        headers: { "content-type": "application/json" },
      }),
    );

    await expect(deleteResource("res-1")).rejects.toThrow("resource delete denied");
  });

  it("surfaces envelope error for resource create non-ok response", async () => {
    const fetchMock = vi.mocked(globalThis.fetch);
    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify({ error: "resource create denied" }), {
        status: 400,
        headers: { "content-type": "application/json" },
      }),
    );

    await expect(createResource({ url: "https://example.com/new" })).rejects.toThrow("resource create denied");
  });

  it("falls back to status text when resource update non-ok body is malformed", async () => {
    const fetchMock = vi.mocked(globalThis.fetch);
    fetchMock.mockResolvedValueOnce(
      new Response("<html>gateway timeout</html>", {
        status: 504,
        headers: { "content-type": "text/html" },
      }),
    );

    await expect(updateResource("res-1", { title: "Updated" })).rejects.toThrow("Request failed (504)");
  });

  it("trims and maps create todo payload fields", async () => {
    const fetchMock = vi.mocked(globalThis.fetch);
    fetchMock.mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          data: {
            id: "todo-1",
            title: "New",
            details: "details",
            status: "open",
            due_at: "2026-04-20T10:30:00Z",
            resource_id: "res-1",
            created_at: "2026-04-18T12:00:00Z",
            updated_at: "2026-04-18T12:00:00Z",
          },
        }),
        {
          status: 200,
          headers: { "content-type": "application/json" },
        },
      ),
    );

    await createTodo({
      title: "  New  ",
      details: "  details  ",
      dueAt: " 2026-04-20T10:30:00Z ",
      resourceId: "  res-1  ",
    });

    const call = fetchMock.mock.calls[0];
    const requestOptions = call?.[1] as RequestInit;
    const requestBody = JSON.parse(String(requestOptions.body)) as Record<string, unknown>;

    expect(requestBody).toEqual({
      title: "New",
      details: "details",
      due_at: "2026-04-20T10:30:00Z",
      resource_id: "res-1",
    });
  });

  it("trims and maps update todo payload fields", async () => {
    const fetchMock = vi.mocked(globalThis.fetch);
    fetchMock.mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          data: {
            id: "todo-1",
            title: "Updated",
            details: "updated details",
            status: "in_progress",
            due_at: "2026-04-20T11:30:00Z",
            resource_id: "res-2",
            created_at: "2026-04-18T12:00:00Z",
            updated_at: "2026-04-18T12:30:00Z",
          },
        }),
        {
          status: 200,
          headers: { "content-type": "application/json" },
        },
      ),
    );

    await updateTodo("todo-1", {
      title: "  Updated  ",
      details: "  updated details  ",
      status: "in_progress",
      dueAt: " 2026-04-20T11:30:00Z ",
      resourceId: "  res-2  ",
    });

    const call = fetchMock.mock.calls[0];
    const requestOptions = call?.[1] as RequestInit;
    const requestBody = JSON.parse(String(requestOptions.body)) as Record<string, unknown>;

    expect(requestBody).toEqual({
      title: "Updated",
      details: "updated details",
      status: "in_progress",
      due_at: "2026-04-20T11:30:00Z",
      resource_id: "res-2",
    });
  });

  it("trims and maps create reminder payload fields", async () => {
    const fetchMock = vi.mocked(globalThis.fetch);
    fetchMock.mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          data: {
            id: "rem-1",
            title: "Reminder",
            message: "details",
            remind_at: "2026-04-21T09:00:00Z",
            status: "scheduled",
            resource_id: "res-1",
            created_at: "2026-04-18T12:00:00Z",
            updated_at: "2026-04-18T12:00:00Z",
          },
        }),
        {
          status: 200,
          headers: { "content-type": "application/json" },
        },
      ),
    );

    await createReminder({
      title: "  Reminder  ",
      message: "  details  ",
      remindAt: " 2026-04-21T09:00:00Z ",
      resourceId: "  res-1  ",
    });

    const call = fetchMock.mock.calls[0];
    const requestOptions = call?.[1] as RequestInit;
    const requestBody = JSON.parse(String(requestOptions.body)) as Record<string, unknown>;

    expect(requestBody).toEqual({
      title: "Reminder",
      message: "details",
      remind_at: "2026-04-21T09:00:00Z",
      resource_id: "res-1",
    });
  });

  it("trims and maps update reminder payload fields", async () => {
    const fetchMock = vi.mocked(globalThis.fetch);
    fetchMock.mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          data: {
            id: "rem-1",
            title: "Reminder",
            message: "updated",
            remind_at: "2026-04-21T10:00:00Z",
            status: "dismissed",
            resource_id: "res-2",
            created_at: "2026-04-18T12:00:00Z",
            updated_at: "2026-04-18T12:30:00Z",
          },
        }),
        {
          status: 200,
          headers: { "content-type": "application/json" },
        },
      ),
    );

    await updateReminder("rem-1", {
      title: "  Reminder  ",
      message: "  updated  ",
      remindAt: " 2026-04-21T10:00:00Z ",
      status: "dismissed",
      resourceId: "  res-2  ",
    });

    const call = fetchMock.mock.calls[0];
    const requestOptions = call?.[1] as RequestInit;
    const requestBody = JSON.parse(String(requestOptions.body)) as Record<string, unknown>;

    expect(requestBody).toEqual({
      title: "Reminder",
      message: "updated",
      remind_at: "2026-04-21T10:00:00Z",
      status: "dismissed",
      resource_id: "res-2",
    });
  });

  it("uses Action and Message fields when sending chat commands", async () => {
    const fetchMock = vi.mocked(globalThis.fetch);
    fetchMock.mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          data: {
            Action: "todo_created",
            Message: "Created todo.",
          },
        }),
        {
          status: 200,
          headers: { "content-type": "application/json" },
        },
      ),
    );

    await expect(sendChatCommand("create todo"))
      .resolves.toEqual({ action: "todo_created", message: "Created todo." });
  });

  it("normalizes resource list rows with mixed key casing", async () => {
    const fetchMock = vi.mocked(globalThis.fetch);
    fetchMock.mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          data: [
            {
              ID: "res-1",
              URL: "https://example.com",
              Host: "example.com",
              Title: "Resource One",
              Summary: "Summary",
              CategoryID: "cat-1",
              CategoryName: "Research",
              UserOverride: true,
              CreatedAt: "2026-04-21T10:00:00Z",
              UpdatedAt: "2026-04-21T10:05:00Z",
            },
          ],
        }),
        {
          status: 200,
          headers: { "content-type": "application/json" },
        },
      ),
    );

    await expect(listResources()).resolves.toEqual([
      {
        id: "res-1",
        url: "https://example.com",
        host: "example.com",
        title: "Resource One",
        summary: "Summary",
        categoryId: "cat-1",
        categoryName: "Research",
        userOverride: true,
        createdAt: "2026-04-21T10:00:00Z",
        updatedAt: "2026-04-21T10:05:00Z",
      },
    ]);
  });

  it("normalizes todo list rows with mixed key casing", async () => {
    const fetchMock = vi.mocked(globalThis.fetch);
    fetchMock.mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          data: [
            {
              ID: "todo-1",
              Title: "Draft roadmap",
              Details: "link task to resource",
              Status: "in_progress",
              DueAt: "2026-04-20T10:30:00Z",
              ResourceID: "res-1",
              CreatedAt: "2026-04-18T12:00:00Z",
              UpdatedAt: "2026-04-18T12:30:00Z",
            },
          ],
        }),
        {
          status: 200,
          headers: { "content-type": "application/json" },
        },
      ),
    );

    await expect(listTodos()).resolves.toEqual([
      {
        id: "todo-1",
        title: "Draft roadmap",
        details: "link task to resource",
        status: "in_progress",
        dueAt: "2026-04-20T10:30:00Z",
        resourceId: "res-1",
        createdAt: "2026-04-18T12:00:00Z",
        updatedAt: "2026-04-18T12:30:00Z",
      },
    ]);
  });

  it("normalizes reminder list rows with mixed key casing", async () => {
    const fetchMock = vi.mocked(globalThis.fetch);
    fetchMock.mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          data: [
            {
              ID: "rem-1",
              Title: "Check task links",
              Message: "verify related resource",
              RemindAt: "2026-04-21T09:00:00Z",
              Status: "scheduled",
              ResourceID: "res-1",
              CreatedAt: "2026-04-18T12:00:00Z",
              UpdatedAt: "2026-04-18T12:00:00Z",
            },
          ],
        }),
        {
          status: 200,
          headers: { "content-type": "application/json" },
        },
      ),
    );

    await expect(listReminders()).resolves.toEqual([
      {
        id: "rem-1",
        title: "Check task links",
        message: "verify related resource",
        remindAt: "2026-04-21T09:00:00Z",
        status: "scheduled",
        resourceId: "res-1",
        createdAt: "2026-04-18T12:00:00Z",
        updatedAt: "2026-04-18T12:00:00Z",
      },
    ]);
  });

  it("omits blank resource fields from create payload", async () => {
    const fetchMock = vi.mocked(globalThis.fetch);
    fetchMock.mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          data: {
            id: "res-1",
            url: "https://example.com",
            host: "example.com",
            title: "Resource One",
            summary: "",
            category_id: "",
            category_name: "",
            user_override: false,
            created_at: "2026-04-21T10:00:00Z",
            updated_at: "2026-04-21T10:05:00Z",
          },
        }),
        {
          status: 200,
          headers: { "content-type": "application/json" },
        },
      ),
    );

    await createResource({
      url: " https://example.com ",
      title: "   ",
      summary: "",
      categoryName: "  ",
    });

    const call = fetchMock.mock.calls[0];
    const requestOptions = call?.[1] as RequestInit;
    const requestBody = JSON.parse(String(requestOptions.body)) as Record<string, unknown>;

    expect(requestBody).toEqual({
      url: "https://example.com",
    });
  });

  it("omits blank resource fields from update payload", async () => {
    const fetchMock = vi.mocked(globalThis.fetch);
    fetchMock.mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          data: {
            id: "res-1",
            url: "https://example.com",
            host: "example.com",
            title: "Resource One",
            summary: "",
            category_id: "",
            category_name: "",
            user_override: false,
            created_at: "2026-04-21T10:00:00Z",
            updated_at: "2026-04-21T10:05:00Z",
          },
        }),
        {
          status: 200,
          headers: { "content-type": "application/json" },
        },
      ),
    );

    await updateResource("res-1", {
      title: "   ",
      summary: "",
      categoryName: "  ",
    });

    const call = fetchMock.mock.calls[0];
    const requestOptions = call?.[1] as RequestInit;
    const requestBody = JSON.parse(String(requestOptions.body)) as Record<string, unknown>;

    expect(requestBody).toEqual({});
  });
});