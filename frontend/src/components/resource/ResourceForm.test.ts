import { describe, expect, it } from "vitest";

import type { ResourceItem } from "../../types";
import { getResourceFormCopy } from "./ResourceForm";

const baseResource: ResourceItem = {
  id: "res-1",
  url: "https://example.com",
  host: "example.com",
  title: "Resource One",
  summary: "Summary",
  categoryId: "cat-1",
  categoryName: "Research",
  userOverride: false,
  createdAt: "2026-04-21T10:00:00Z",
  updatedAt: "2026-04-21T10:05:00Z",
};

describe("ResourceForm helpers", () => {
  it("returns add copy when no resource is selected", () => {
    expect(getResourceFormCopy(null)).toEqual({
      heading: "Add Resource",
      subheading: "Create a new resource node.",
    });
  });

  it("returns edit copy with the resource title", () => {
    expect(getResourceFormCopy(baseResource)).toEqual({
      heading: "Edit Resource",
      subheading: "Selected: Resource One",
    });
  });

  it("falls back to url when the title is blank", () => {
    const resource = { ...baseResource, title: "" };

    expect(getResourceFormCopy(resource)).toEqual({
      heading: "Edit Resource",
      subheading: "Selected: https://example.com",
    });
  });
});
