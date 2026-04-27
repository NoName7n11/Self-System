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
});