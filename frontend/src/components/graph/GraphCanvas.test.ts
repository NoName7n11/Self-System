import { describe, expect, it } from "vitest";

import type { ResourceItem } from "../../types";
import {
  GRAPH_LOD_NODE_THRESHOLD,
  buildGraph,
  categoryColor,
  getGraphRenderConfig,
  getNodeColor,
  getNodeLabel,
  toFinite,
  type GraphNode,
} from "./GraphCanvas";

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

describe("GraphCanvas helpers", () => {
  it("creates deterministic category colors", () => {
    expect(categoryColor("Atlas")).toBe("hsl(226 78% 63%)");
  });

  it("returns zero for non-finite graph coordinates", () => {
    expect(toFinite(undefined)).toBe(0);
    expect(toFinite(Number.NaN)).toBe(0);
    expect(toFinite(4)).toBe(4);
  });

  it("builds node labels for resource and category nodes", () => {
    expect(
      getNodeLabel({
        id: "res-1",
        label: "Atlas",
        kind: "resource",
        color: "#fff",
        val: 6,
        categoryName: "Research",
      }),
    ).toBe("Atlas (Research)");

    expect(
      getNodeLabel({
        id: "category:research",
        label: "Research",
        kind: "category",
        color: "#fff",
        val: 10,
      }),
    ).toBe("Research category");
  });

  it("highlights the selected resource node", () => {
    const node: GraphNode = {
      id: "res-1",
      label: "Atlas",
      kind: "resource",
      color: "#59d6ff",
      val: 6,
      resourceId: "res-1",
    };

    expect(getNodeColor(node, "res-1")).toBe("#c5ff4d");
    expect(getNodeColor(node, null)).toBe("#59d6ff");
  });

  it("builds graph nodes and links from resource data", () => {
    const resources = [
      buildResource({
        id: "res-1",
        title: "Atlas One",
        categoryName: "Research",
        userOverride: false,
      }),
      buildResource({
        id: "res-2",
        title: " ",
        host: "b.com",
        url: "https://b.com",
        categoryName: "research",
        userOverride: true,
      }),
      buildResource({
        id: "res-3",
        title: " ",
        host: " ",
        url: " ",
        categoryName: "   ",
        userOverride: false,
      }),
    ];

    const graph = buildGraph(resources);
    expect(graph.nodes).toHaveLength(5);
    expect(graph.links).toHaveLength(3);

    const researchNode = graph.nodes.find((node) => node.id === "category:research");
    expect(researchNode?.label).toBe("Research");
    expect(researchNode?.kind).toBe("category");
    expect(researchNode?.val).toBeCloseTo(23.2, 3);

    const unsortedNode = graph.nodes.find((node) => node.id === "category:unsorted");
    expect(unsortedNode?.label).toBe("Unsorted");
    expect(unsortedNode?.val).toBeCloseTo(20.6, 3);

    const overrideNode = graph.nodes.find((node) => node.id === "res-2");
    expect(overrideNode?.label).toBe("b.com");
    expect(overrideNode?.color).toBe("#ffcc66");
    expect(overrideNode?.val).toBe(9);

    const defaultNode = graph.nodes.find((node) => node.id === "res-3");
    expect(defaultNode?.label).toBe("Resource");
    expect(defaultNode?.color).toBe("#59d6ff");
    expect(defaultNode?.val).toBe(6);

    const overrideLink = graph.links.find((link) => link.source === "res-2");
    expect(overrideLink?.target).toBe("category:research");
    expect(overrideLink?.color).toBe("rgba(255, 204, 102, 0.56)");
    expect(overrideLink?.strength).toBe(1.6);
  });
});

describe("getGraphRenderConfig (Change 13 WS4 LOD)", () => {
  it("uses full-detail rendering below the LOD threshold", () => {
    const config = getGraphRenderConfig(GRAPH_LOD_NODE_THRESHOLD);
    expect(config.forceMode).toBeNull();
    expect(config.showLabels).toBe(true);
    expect(config.linkDirectionalParticles).toBe(1);
    expect(config.cooldownTicks).toBe(120);
  });

  it("drops to 2D and trims per-frame work above the LOD threshold", () => {
    const config = getGraphRenderConfig(GRAPH_LOD_NODE_THRESHOLD + 1);
    expect(config.forceMode).toBe("2d");
    expect(config.showLabels).toBe(false);
    expect(config.linkDirectionalParticles).toBe(0);
    expect(config.cooldownTicks).toBe(60);
  });

  it("keeps degraded rendering stable for a synthetic 10k-node graph", () => {
    const resources: ResourceItem[] = Array.from({ length: 10000 }, (_, index) =>
      buildResource({
        id: `res-${index}`,
        title: `Resource ${index}`,
        categoryName: `Category ${index % 25}`,
      }),
    );

    const graph = buildGraph(resources);
    expect(graph.nodes.length).toBeGreaterThan(10000);

    const config = getGraphRenderConfig(graph.nodes.length);
    expect(config.forceMode).toBe("2d");
    expect(config.showLabels).toBe(false);
    expect(config.linkDirectionalParticles).toBe(0);
    expect(config.cooldownTicks).toBe(60);
  });
});
