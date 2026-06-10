// @vitest-environment jsdom
import React from "react";

import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";

import ResourceForm from "../../../src/components/resource/ResourceForm";
import { useResourceStore } from "../../../src/stores/useResourceStore";
import type { ResourceDraft, ResourceItem } from "../../../src/types";

const defaultDraft: ResourceDraft = {
  url: "",
  title: "",
  summary: "",
  categoryName: "",
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

function resetResourceStore() {
  useResourceStore.setState({
    resources: [],
    isLoading: false,
    error: null,
    selectedResourceId: null,
    draft: { ...defaultDraft },
  });
}

describe("ResourceForm integration", () => {
  beforeEach(() => {
    resetResourceStore();
  });

  afterEach(() => {
    cleanup();
  });

  it("shows validation error when URL is missing", () => {
    render(<ResourceForm />);

    fireEvent.click(screen.getByRole("button", { name: "Add As New" }));

    expect(screen.getByText("A URL is required to create a resource.")).toBeTruthy();
  });

  it("enables update when a resource is selected", () => {
    render(<ResourceForm />);

    const updateButton = screen.getByRole("button", { name: "Update Selected" }) as HTMLButtonElement;
    expect(updateButton.disabled).toBe(true);

    act(() => {
      useResourceStore.setState({
        resources: [seedResource],
        selectedResourceId: seedResource.id,
        draft: {
          url: seedResource.url,
          title: seedResource.title,
          summary: seedResource.summary,
          categoryName: seedResource.categoryName,
        },
      });
    });

    const enabledButton = screen.getByRole("button", { name: "Update Selected" }) as HTMLButtonElement;
    expect(enabledButton.disabled).toBe(false);
  });
});
