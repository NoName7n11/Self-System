// @vitest-environment jsdom
import React from "react";

import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";

import ResourceList from "../../../src/components/resource/ResourceList";
import { useResourceStore } from "../../../src/stores/useResourceStore";
import type { ResourceDraft, ResourceFilters, ResourceItem } from "../../../src/types";

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

const seedResource: ResourceItem = {
  id: "res-1",
  url: "https://example.com/seed",
  host: "example.com",
  title: "Seed Resource",
  summary: "Seeded summary",
  categoryId: "cat-1",
  categoryName: "Research",
  userOverride: false,
  createdAt: "2026-05-01T12:00:00Z",
  updatedAt: "2026-05-01T12:00:00Z",
};

const overrideResource: ResourceItem = {
  ...seedResource,
  id: "res-2",
  title: "Override Resource",
  userOverride: true,
};

function resetResourceStore() {
  useResourceStore.setState({
    resources: [],
    isLoading: false,
    error: null,
    selectedResourceId: null,
    filters: { ...defaultFilters },
    draft: { ...defaultDraft },
  });
}

describe("ResourceList integration", () => {
  beforeEach(() => {
    resetResourceStore();
  });

  afterEach(() => {
    cleanup();
  });

  it("shows empty state when there are no resources", () => {
    render(<ResourceList resources={[]} />);

    expect(screen.getByText("No resources match the current filters.")).toBeTruthy();
  });

  it("shows override counts in the header", () => {
    render(<ResourceList resources={[seedResource, overrideResource]} />);

    expect(screen.getByText("2 visible, 1 override-tagged")).toBeTruthy();
  });

  it("selects a resource when clicked", () => {
    useResourceStore.setState({ resources: [seedResource] });

    render(<ResourceList resources={[seedResource]} />);

    const row = screen.getByText("Seed Resource").closest("button");
    expect(row).toBeTruthy();
    const rowButton = row as HTMLButtonElement;

    expect(rowButton.className.includes("is-selected")).toBe(false);

    fireEvent.click(rowButton);

    const selectedRow = screen.getByText("Seed Resource").closest("button") as HTMLButtonElement | null;
    expect(selectedRow).toBeTruthy();
    expect((selectedRow as HTMLButtonElement).className.includes("is-selected")).toBe(true);
  });
});
