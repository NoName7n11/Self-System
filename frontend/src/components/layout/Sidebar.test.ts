import { describe, expect, it } from "vitest";

import type { ResourceItem } from "../../types";
import { deriveFavorites, deriveRecents } from "./Sidebar";

const baseResource: ResourceItem = {
  id: "res-0",
  url: "https://example.com",
  host: "example.com",
  title: "Resource",
  summary: "Summary",
  categoryId: "cat-0",
  categoryName: "Research",
  userOverride: false,
  createdAt: "2026-04-01T00:00:00Z",
  updatedAt: "2026-04-01T00:00:00Z",
};

function buildResource(overrides: Partial<ResourceItem>): ResourceItem {
  return {
    ...baseResource,
    ...overrides,
  };
}

describe("Sidebar helpers", () => {
  it("derives top favorite categories with counts", () => {
    const resources = [
      buildResource({ id: "res-1", categoryName: "Ops" }),
      buildResource({ id: "res-2", categoryName: "Ops" }),
      buildResource({ id: "res-3", categoryName: "Ops" }),
      buildResource({ id: "res-4", categoryName: "Research" }),
      buildResource({ id: "res-5", categoryName: "Research" }),
      buildResource({ id: "res-6", categoryName: "   " }),
    ];

    const favorites = deriveFavorites(resources);
    expect(favorites).toEqual([
      ["Ops", 3],
      ["Research", 2],
      ["Unsorted", 1],
    ]);
  });

  it("derives recent resources ordered by created date", () => {
    const resources = [
      buildResource({ id: "res-1", createdAt: "2026-04-01T00:00:00Z" }),
      buildResource({ id: "res-2", createdAt: "2026-05-01T00:00:00Z" }),
      buildResource({ id: "res-3", createdAt: "2026-03-01T00:00:00Z" }),
      buildResource({ id: "res-4", createdAt: "2026-04-15T00:00:00Z" }),
      buildResource({ id: "res-5", createdAt: "2026-02-01T00:00:00Z" }),
    ];

    const recents = deriveRecents(resources);
    expect(recents.map((item) => item.id)).toEqual(["res-2", "res-4", "res-1", "res-3"]);
  });
});
