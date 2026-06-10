import { describe, expect, it } from "vitest";

import type { ResourceItem } from "../../types";
import { deriveCategoryOptions } from "./GraphControls";

const baseResource: ResourceItem = {
  id: "res-0",
  url: "https://example.com",
  host: "example.com",
  title: "Resource",
  summary: "Summary",
  categoryId: "cat-0",
  categoryName: "Research",
  userOverride: false,
  createdAt: "2026-04-21T10:00:00Z",
  updatedAt: "2026-04-21T10:05:00Z",
};

function buildResource(overrides: Partial<ResourceItem>): ResourceItem {
  return {
    ...baseResource,
    ...overrides,
  };
}

describe("deriveCategoryOptions", () => {
  it("trims, dedupes, and sorts category options", () => {
    const resources = [
      buildResource({ id: "res-1", categoryName: " Research " }),
      buildResource({ id: "res-2", categoryName: "Planning" }),
      buildResource({ id: "res-3", categoryName: "Research" }),
      buildResource({ id: "res-4", categoryName: "" }),
      buildResource({ id: "res-5", categoryName: "   " }),
    ];

    const categories = deriveCategoryOptions(resources);

    expect(categories).toEqual(["Planning", "Research"]);
  });

  it("returns an empty array when no categories are present", () => {
    const resources = [
      buildResource({ id: "res-1", categoryName: "" }),
      buildResource({ id: "res-2", categoryName: "   " }),
    ];

    expect(deriveCategoryOptions(resources)).toEqual([]);
  });
});
