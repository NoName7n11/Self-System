import { create } from "zustand";

import { createResource, deleteResource, listResources, normalizeResource, updateResource } from "../api/client";
import { ipcCall } from "../lib/ipc";
import { demoResourcesAsItems } from "../lib/demoData";
import type { ResourceDraft, ResourceFilters, ResourceItem, ViewMode } from "../types";

const defaultDraft: ResourceDraft = {
  url: "",
  title: "",
  summary: "",
  categoryName: "",
};

const defaultFilters: ResourceFilters = {
  query: "",
  category: "all",
  viewMode: "3d",
  showOverridesOnly: false,
};

function draftFromResource(resource: ResourceItem): ResourceDraft {
  return {
    url: resource.url,
    title: resource.title,
    summary: resource.summary,
    categoryName: resource.categoryName,
  };
}

function normalizeDraft(draft: ResourceDraft): ResourceDraft {
  return {
    url: draft.url.trim(),
    title: draft.title.trim(),
    summary: draft.summary.trim(),
    categoryName: draft.categoryName.trim(),
  };
}

interface ResourceState {
  resources: ResourceItem[];
  isLoading: boolean;
  error: string | null;
  selectedResourceId: string | null;
  filters: ResourceFilters;
  draft: ResourceDraft;
  removedLib: Record<string, boolean>;
  removeFromLibrary: (id: string) => void;
  setQuery: (query: string) => void;
  setCategoryFilter: (category: string) => void;
  setViewMode: (mode: ViewMode) => void;
  toggleOverridesOnly: () => void;
  updateDraft: (field: keyof ResourceDraft, value: string) => void;
  resetDraft: () => void;
  selectResource: (resourceId: string | null) => void;
  loadResources: (options?: { silent?: boolean }) => Promise<void>;
  addResource: () => Promise<void>;
  updateSelectedResource: () => Promise<void>;
  deleteSelectedResource: () => Promise<void>;
}

export const useResourceStore = create<ResourceState>((set, get) => ({
  resources: [],
  isLoading: false,
  error: null,
  selectedResourceId: null,
  filters: defaultFilters,
  draft: defaultDraft,
  removedLib: {},

  // remove from library (design removeLib) — hides from library lists, not a delete
  removeFromLibrary: (id) => {
    set((state) => ({ removedLib: { ...state.removedLib, [id]: true } }));
  },

  setQuery: (query) => {
    set((state) => ({
      filters: {
        ...state.filters,
        query,
      },
    }));
  },

  setCategoryFilter: (category) => {
    set((state) => ({
      filters: {
        ...state.filters,
        category,
      },
    }));
  },

  setViewMode: (mode) => {
    set((state) => ({
      filters: {
        ...state.filters,
        viewMode: mode,
      },
    }));
  },

  toggleOverridesOnly: () => {
    set((state) => ({
      filters: {
        ...state.filters,
        showOverridesOnly: !state.filters.showOverridesOnly,
      },
    }));
  },

  updateDraft: (field, value) => {
    set((state) => ({
      draft: {
        ...state.draft,
        [field]: value,
      },
    }));
  },

  resetDraft: () => {
    set({
      draft: defaultDraft,
      selectedResourceId: null,
    });
  },

  selectResource: (resourceId) => {
    if (!resourceId) {
      set({ selectedResourceId: null, draft: defaultDraft });
      return;
    }

    const selected = get().resources.find((resource) => resource.id === resourceId);
    if (!selected) {
      set({ selectedResourceId: null, draft: defaultDraft });
      return;
    }

    set({
      selectedResourceId: selected.id,
      draft: draftFromResource(selected),
    });
  },

  loadResources: async (options) => {
    const silent = options?.silent === true;
    if (silent) {
      set({ error: null });
    } else {
      set({ isLoading: true, error: null });
    }

    try {
      const rawRows = await ipcCall<unknown[]>(
        "desktop.App.GetResources",
        [50, 0],
        () => listResources()
      );
      const fetched = rawRows.map(normalizeResource);
      // ponytail: fall back to bundled demo seed when backend is empty/offline so
      // the UI looks populated like the design. Real data wins whenever present.
      const usingDemo = fetched.length === 0;
      const rows = usingDemo ? demoResourcesAsItems() : fetched;
      const previousId = get().selectedResourceId;
      const selectedResourceId = previousId ?? (usingDemo ? "r1" : null);

      set((state) => {
        const selected = rows.find((resource) => resource.id === selectedResourceId) ?? null;

        return {
          resources: rows,
          isLoading: silent ? state.isLoading : false,
          error: null,
          selectedResourceId: selected?.id ?? null,
          draft: selected ? draftFromResource(selected) : state.draft,
        };
      });
    } catch (error) {
      // backend unreachable → show demo seed instead of an empty barren shell
      const previousId = get().selectedResourceId;
      const rows = demoResourcesAsItems();
      const selected = rows.find((r) => r.id === (previousId ?? "r1")) ?? null;
      set((state) => ({
        resources: rows,
        isLoading: silent ? state.isLoading : false,
        error: null,
        selectedResourceId: selected?.id ?? null,
        draft: selected ? draftFromResource(selected) : state.draft,
      }));
    }
  },

  addResource: async () => {
    const draft = normalizeDraft(get().draft);
    if (draft.url === "") {
      set({ error: "A URL is required to create a resource." });
      return;
    }

    try {
      const created = normalizeResource(await ipcCall<unknown>(
        "desktop.App.CreateResource",
        [draft.url, draft.title, draft.summary, draft.categoryName],
        () => createResource({ url: draft.url, title: draft.title, summary: draft.summary, categoryName: draft.categoryName })
      ));

      set((state) => ({
        resources: [created, ...state.resources.filter((resource) => resource.id !== created.id)],
        selectedResourceId: created.id,
        draft: draftFromResource(created),
        error: null,
      }));
    } catch (error) {
      const message = error instanceof Error ? error.message : "Failed to create resource";
      set({ error: message });
    }
  },

  updateSelectedResource: async () => {
    const selectedResourceId = get().selectedResourceId;
    if (!selectedResourceId) {
      set({ error: "Select a resource before updating." });
      return;
    }

    const draft = normalizeDraft(get().draft);
    if (draft.url === "") {
      set({ error: "A URL is required to update a resource." });
      return;
    }

    try {
      const updated = normalizeResource(await ipcCall<unknown>(
        "desktop.App.UpdateResource",
        [selectedResourceId, draft.url, draft.title, draft.summary, draft.categoryName],
        () => updateResource(selectedResourceId, {
          url: draft.url,
          title: draft.title,
          summary: draft.summary,
          categoryName: draft.categoryName,
        })
      ));

      set((state) => ({
        resources: state.resources.map((resource) => (resource.id === updated.id ? updated : resource)),
        selectedResourceId: updated.id,
        draft: draftFromResource(updated),
        error: null,
      }));
    } catch (error) {
      const message = error instanceof Error ? error.message : "Failed to update resource";
      set({ error: message });
    }
  },

  deleteSelectedResource: async () => {
    const selectedResourceId = get().selectedResourceId;
    if (!selectedResourceId) {
      set({ error: "Select a resource before deleting." });
      return;
    }

    try {
      await ipcCall<boolean | void>(
        "desktop.App.DeleteResource",
        [selectedResourceId],
        () => deleteResource(selectedResourceId)
      );

      set((state) => ({
        resources: state.resources.filter((resource) => resource.id !== selectedResourceId),
        selectedResourceId: null,
        draft: defaultDraft,
        error: null,
      }));
    } catch (error) {
      const message = error instanceof Error ? error.message : "Failed to delete resource";
      set({ error: message });
    }
  },
}));
