import { beforeEach, describe, expect, it, vi } from "vitest";

import { createResource, deleteResource, listResources, updateResource } from "../api/client";
import { useResourceStore } from "./useResourceStore";

vi.mock("../api/client", () => ({
  listResources: vi.fn(),
  createResource: vi.fn(),
  updateResource: vi.fn(),
  deleteResource: vi.fn(),
}));

function resetResourceStore() {
  useResourceStore.setState({
    resources: [],
    isLoading: false,
    error: null,
    selectedResourceId: null,
    filters: {
      query: "",
      category: "all",
      viewMode: "3d",
      showOverridesOnly: false,
    },
    draft: {
      url: "",
      title: "",
      summary: "",
      categoryName: "",
    },
  });
}

function seedSelectedResource() {
  useResourceStore.setState({
    resources: [
      {
        id: "res-1",
        url: "https://example.com",
        host: "example.com",
        title: "Resource One",
        summary: "Summary",
        categoryId: "cat-1",
        categoryName: "Research",
        userOverride: false,
        createdAt: "2026-04-20T10:00:00.000Z",
        updatedAt: "2026-04-20T10:00:00.000Z",
      },
    ],
    selectedResourceId: "res-1",
    draft: {
      url: "https://example.com",
      title: "Resource One",
      summary: "Summary",
      categoryName: "Research",
    },
    error: null,
  });
}

function seedSecondaryDraft() {
  useResourceStore.setState({
    draft: {
      url: "https://example.com/new",
      title: "New Resource",
      summary: "Create draft summary",
      categoryName: "Research",
    },
  });
}

describe("useResourceStore", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetResourceStore();

    vi.mocked(listResources).mockResolvedValue([]);
    vi.mocked(createResource).mockResolvedValue({
      id: "res-1",
      url: "https://example.com",
      host: "example.com",
      title: "Resource One",
      summary: "Summary",
      categoryId: "cat-1",
      categoryName: "Research",
      userOverride: false,
      createdAt: "2026-04-20T10:00:00.000Z",
      updatedAt: "2026-04-20T10:00:00.000Z",
    });
    vi.mocked(updateResource).mockResolvedValue({
      id: "res-1",
      url: "https://example.com",
      host: "example.com",
      title: "Resource One Updated",
      summary: "Updated Summary",
      categoryId: "cat-1",
      categoryName: "Research",
      userOverride: false,
      createdAt: "2026-04-20T10:00:00.000Z",
      updatedAt: "2026-04-21T08:30:00.000Z",
    });
    vi.mocked(deleteResource).mockResolvedValue(undefined);
  });

  it("creates resource with normalized payload and selects it", async () => {
    vi.mocked(createResource).mockResolvedValueOnce({
      id: "res-2",
      url: "https://example.com/new",
      host: "example.com",
      title: "New Resource",
      summary: "Create draft summary",
      categoryId: "cat-2",
      categoryName: "Research",
      userOverride: false,
      createdAt: "2026-04-21T09:00:00.000Z",
      updatedAt: "2026-04-21T09:00:00.000Z",
    });

    useResourceStore.getState().updateDraft("url", "  https://example.com/new  ");
    useResourceStore.getState().updateDraft("title", "  New Resource  ");
    useResourceStore.getState().updateDraft("summary", "  Create draft summary  ");
    useResourceStore.getState().updateDraft("categoryName", "  Research  ");

    await useResourceStore.getState().addResource();

    expect(createResource).toHaveBeenCalledWith({
      url: "https://example.com/new",
      title: "New Resource",
      summary: "Create draft summary",
      categoryName: "Research",
    });

    const state = useResourceStore.getState();
    expect(state.resources).toHaveLength(1);
    expect(state.resources[0]?.id).toBe("res-2");
    expect(state.selectedResourceId).toBe("res-2");
    expect(state.draft).toEqual({
      url: "https://example.com/new",
      title: "New Resource",
      summary: "Create draft summary",
      categoryName: "Research",
    });
    expect(state.error).toBeNull();
  });

  it("sets create validation error when url is blank", async () => {
    useResourceStore.getState().updateDraft("url", "   ");
    useResourceStore.getState().updateDraft("title", "Missing URL");

    await useResourceStore.getState().addResource();

    expect(createResource).not.toHaveBeenCalled();
    expect(useResourceStore.getState().error).toBe("A URL is required to create a resource.");
  });

  it("sets create error and preserves existing state", async () => {
    seedSelectedResource();
    seedSecondaryDraft();
    vi.mocked(createResource).mockRejectedValueOnce(new Error("mock create resource error"));

    await useResourceStore.getState().addResource();

    const state = useResourceStore.getState();
    expect(state.error).toBe("mock create resource error");
    expect(state.resources).toHaveLength(1);
    expect(state.resources[0]?.id).toBe("res-1");
    expect(state.selectedResourceId).toBe("res-1");
  });

  it("updates selected resource and keeps selection", async () => {
    seedSelectedResource();

    useResourceStore.getState().updateDraft("title", "Resource One Updated");
    useResourceStore.getState().updateDraft("summary", "Updated Summary");
    useResourceStore.getState().updateDraft("categoryName", "Research");

    await useResourceStore.getState().updateSelectedResource();

    expect(updateResource).toHaveBeenCalledWith(
      "res-1",
      expect.objectContaining({
        title: "Resource One Updated",
        summary: "Updated Summary",
        categoryName: "Research",
      }),
    );

    const state = useResourceStore.getState();
    expect(state.resources).toHaveLength(1);
    expect(state.resources[0]?.title).toBe("Resource One Updated");
    expect(state.selectedResourceId).toBe("res-1");
    expect(state.error).toBeNull();
  });

  it("sets update validation error when no resource is selected", async () => {
    seedSecondaryDraft();

    await useResourceStore.getState().updateSelectedResource();

    expect(updateResource).not.toHaveBeenCalled();
    expect(useResourceStore.getState().error).toBe("Select a resource before updating.");
  });

  it("sets update validation error when selected draft url is blank", async () => {
    seedSelectedResource();
    useResourceStore.getState().updateDraft("url", "   ");

    await useResourceStore.getState().updateSelectedResource();

    expect(updateResource).not.toHaveBeenCalled();
    expect(useResourceStore.getState().error).toBe("A URL is required to update a resource.");
  });

  it("sets update error and preserves existing resource", async () => {
    seedSelectedResource();
    useResourceStore.getState().updateDraft("title", "Should not persist");
    vi.mocked(updateResource).mockRejectedValueOnce(new Error("mock update resource error"));

    await useResourceStore.getState().updateSelectedResource();

    expect(updateResource).toHaveBeenCalledWith(
      "res-1",
      expect.objectContaining({
        title: "Should not persist",
      }),
    );

    const state = useResourceStore.getState();
    expect(state.error).toBe("mock update resource error");
    expect(state.resources).toHaveLength(1);
    expect(state.resources[0]?.title).toBe("Resource One");
    expect(state.selectedResourceId).toBe("res-1");
  });

  it("keeps selected resource and draft when silent refresh fails", async () => {
    seedSelectedResource();
    vi.mocked(listResources).mockRejectedValueOnce(new Error("mock load resources error"));

    await useResourceStore.getState().loadResources({ silent: true });

    const state = useResourceStore.getState();
    expect(state.resources).toHaveLength(1);
    expect(state.resources[0]?.id).toBe("res-1");
    expect(state.selectedResourceId).toBe("res-1");
    expect(state.draft.title).toBe("Resource One");
    expect(state.error).toBe("mock load resources error");
  });

  it("sets error when resource list fails (non-silent)", async () => {
    // non-silent load should surface errors and clear selection when no data returned
    seedSelectedResource();
    vi.mocked(listResources).mockRejectedValueOnce(new Error("mock load resources failure"));

    await useResourceStore.getState().loadResources();

    const state = useResourceStore.getState();
    expect(state.error).toBe("mock load resources failure");
    // store preserves existing resources on list failure
    expect(state.resources).toHaveLength(1);
    expect(state.selectedResourceId).toBe("res-1");
  });

  it("retains selected resource and refreshes draft on silent refresh success", async () => {
    seedSelectedResource();
    vi.mocked(listResources).mockResolvedValueOnce([
      {
        id: "res-1",
        url: "https://example.com",
        host: "example.com",
        title: "Resource One From Server",
        summary: "Server Summary",
        categoryId: "cat-1",
        categoryName: "Research",
        userOverride: false,
        createdAt: "2026-04-20T10:00:00.000Z",
        updatedAt: "2026-04-22T07:00:00.000Z",
      },
    ]);

    await useResourceStore.getState().loadResources({ silent: true });

    const state = useResourceStore.getState();
    expect(state.resources).toHaveLength(1);
    expect(state.resources[0]?.title).toBe("Resource One From Server");
    expect(state.selectedResourceId).toBe("res-1");
    expect(state.draft).toEqual({
      url: "https://example.com",
      title: "Resource One From Server",
      summary: "Server Summary",
      categoryName: "Research",
    });
    expect(state.error).toBeNull();
  });

  it("clears missing selected id but preserves draft on silent refresh success", async () => {
    seedSelectedResource();
    vi.mocked(listResources).mockResolvedValueOnce([
      {
        id: "res-2",
        url: "https://example.org",
        host: "example.org",
        title: "Replacement Resource",
        summary: "New list payload",
        categoryId: "cat-2",
        categoryName: "Planning",
        userOverride: false,
        createdAt: "2026-04-22T07:00:00.000Z",
        updatedAt: "2026-04-22T07:00:00.000Z",
      },
    ]);

    await useResourceStore.getState().loadResources({ silent: true });

    const state = useResourceStore.getState();
    expect(state.resources).toHaveLength(1);
    expect(state.resources[0]?.id).toBe("res-2");
    expect(state.selectedResourceId).toBeNull();
    expect(state.draft.title).toBe("Resource One");
    expect(state.error).toBeNull();
  });

  it("deletes selected resource and resets draft", async () => {
    seedSelectedResource();

    await useResourceStore.getState().deleteSelectedResource();

    expect(deleteResource).toHaveBeenCalledWith("res-1");

    const state = useResourceStore.getState();
    expect(state.resources).toHaveLength(0);
    expect(state.selectedResourceId).toBeNull();
    expect(state.draft).toEqual({
      url: "",
      title: "",
      summary: "",
      categoryName: "",
    });
    expect(state.error).toBeNull();
  });

  it("sets delete validation error when no resource is selected", async () => {
    await useResourceStore.getState().deleteSelectedResource();

    expect(deleteResource).not.toHaveBeenCalled();
    expect(useResourceStore.getState().error).toBe("Select a resource before deleting.");
  });

  it("sets delete error and keeps selected resource", async () => {
    seedSelectedResource();
    vi.mocked(deleteResource).mockRejectedValueOnce(new Error("mock delete resource error"));

    await useResourceStore.getState().deleteSelectedResource();

    const state = useResourceStore.getState();
    expect(state.error).toBe("mock delete resource error");
    expect(state.resources).toHaveLength(1);
    expect(state.selectedResourceId).toBe("res-1");
  });
});
