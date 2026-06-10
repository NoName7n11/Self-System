import { describe, expect, it } from "vitest";

import type { ResourceFilters, ResourceItem } from "../types";
import { filterResources } from "./useFilteredResources";

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

describe("filterResources", () => {
  it("matches query across title, summary, url, and category", () => {
    const resources = [
      buildResource({
        id: "res-title",
        title: "Atlas Plan",
        summary: "",
        url: "https://example.com/alpha",
        categoryName: "Research",
      }),
      buildResource({
        id: "res-summary",
        title: "Bravo",
        summary: "Atlas notes",
        url: "https://example.com/bravo",
        categoryName: "Planning",
      }),
      buildResource({
        id: "res-url",
        title: "Charlie",
        summary: "",
        url: "https://atlas.dev/resource",
        categoryName: "Operations",
      }),
      buildResource({
        id: "res-category",
        title: "Delta",
        summary: "",
        url: "https://example.com/delta",
        categoryName: "Atlas",
      }),
    ];

    const filters: ResourceFilters = {
      query: "  ATLAS  ",
      category: "all",
      viewMode: "3d",
      showOverridesOnly: false,
    };

    const result = filterResources(resources, filters);
    expect(result.map((resource) => resource.id)).toEqual([
      "res-title",
      "res-summary",
      "res-url",
      "res-category",
    ]);
  });

  it("matches category filters with trim and case-insensitive rules", () => {
    const resources = [
      buildResource({ id: "res-1", categoryName: "Research" }),
      buildResource({ id: "res-2", categoryName: "Planning" }),
    ];

    const filters: ResourceFilters = {
      query: "",
      category: "  PLANNING ",
      viewMode: "3d",
      showOverridesOnly: false,
    };

    const result = filterResources(resources, filters);
    expect(result.map((resource) => resource.id)).toEqual(["res-2"]);
  });

  it("filters to override-only resources when enabled", () => {
    const resources = [
      buildResource({ id: "res-1", userOverride: true }),
      buildResource({ id: "res-2", userOverride: false }),
    ];

    const filters: ResourceFilters = {
      query: "",
      category: "all",
      viewMode: "3d",
      showOverridesOnly: true,
    };

    const result = filterResources(resources, filters);
    expect(result.map((resource) => resource.id)).toEqual(["res-1"]);
  });

  it("combines query and category filters", () => {
    const resources = [
      buildResource({
        id: "res-1",
        title: "Atlas Plan",
        categoryName: "Research",
      }),
      buildResource({
        id: "res-2",
        title: "Atlas Plan",
        categoryName: "Planning",
      }),
    ];

    const filters: ResourceFilters = {
      query: "atlas",
      category: "research",
      viewMode: "2d",
      showOverridesOnly: false,
    };

    const result = filterResources(resources, filters);
    expect(result.map((resource) => resource.id)).toEqual(["res-1"]);
  });
});
