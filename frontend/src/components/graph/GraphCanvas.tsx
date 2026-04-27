import { Suspense, lazy, useEffect, useMemo, useRef } from "react";

import { useResourceStore } from "../../stores/useResourceStore";
import type { ResourceItem } from "../../types";

interface GraphCanvasProps {
  resources: ResourceItem[];
}

interface GraphNode {
  id: string;
  label: string;
  kind: "resource" | "category";
  color: string;
  val: number;
  categoryName?: string;
  resourceId?: string;
  userOverride?: boolean;
  x?: number;
  y?: number;
  z?: number;
}

interface GraphLink {
  source: string;
  target: string;
  color: string;
  strength: number;
}

interface GraphData {
  nodes: GraphNode[];
  links: GraphLink[];
}

const ForceGraph2D = lazy(() => import("react-force-graph-2d"));
const ForceGraph3D = lazy(() => import("react-force-graph-3d"));

function hashSeed(value: string): number {
  let seed = 0;
  for (let index = 0; index < value.length; index += 1) {
    seed = (seed * 31 + value.charCodeAt(index)) >>> 0;
  }
  return seed;
}

function categoryColor(name: string): string {
  const hue = 175 + (hashSeed(name) % 120);
  return `hsl(${hue} 78% 63%)`;
}

function buildGraph(resources: ResourceItem[]): GraphData {
  const nodes: GraphNode[] = [];
  const links: GraphLink[] = [];
  const categoryIdByName = new Map<string, string>();
  const categoryWeight = new Map<string, number>();

  for (const resource of resources) {
    const categoryName = resource.categoryName.trim() || "Unsorted";
    const normalized = categoryName.toLowerCase();

    let categoryId = categoryIdByName.get(normalized);
    if (!categoryId) {
      categoryId = `category:${normalized}`;
      categoryIdByName.set(normalized, categoryId);
      categoryWeight.set(categoryId, 0);

      nodes.push({
        id: categoryId,
        label: categoryName,
        kind: "category",
        color: categoryColor(categoryName),
        val: 24,
        categoryName,
      });
    }

    const title = resource.title.trim() || resource.host.trim() || resource.url.trim() || "Resource";
    nodes.push({
      id: resource.id,
      label: title,
      kind: "resource",
      color: resource.userOverride ? "#ffcc66" : "#59d6ff",
      val: resource.userOverride ? 9 : 6,
      categoryName,
      resourceId: resource.id,
      userOverride: resource.userOverride,
    });

    links.push({
      source: resource.id,
      target: categoryId,
      color: resource.userOverride ? "rgba(255, 204, 102, 0.56)" : "rgba(89, 214, 255, 0.34)",
      strength: resource.userOverride ? 1.6 : 1,
    });

    categoryWeight.set(categoryId, (categoryWeight.get(categoryId) ?? 0) + 1);
  }

  for (const node of nodes) {
    if (node.kind !== "category") {
      continue;
    }
    const id = node.id;
    const weight = categoryWeight.get(id) ?? 0;
    node.val = 18 + Math.min(14, weight * 2.6);
  }

  return { nodes, links };
}

function toFinite(value: number | undefined): number {
  if (typeof value !== "number" || !Number.isFinite(value)) {
    return 0;
  }
  return value;
}

export default function GraphCanvas({ resources }: GraphCanvasProps) {
  const viewMode = useResourceStore((state) => state.filters.viewMode);
  const selectedResourceId = useResourceStore((state) => state.selectedResourceId);
  const selectResource = useResourceStore((state) => state.selectResource);
  const setCategoryFilter = useResourceStore((state) => state.setCategoryFilter);

  const graph2DRef = useRef<any>(null);
  const graph3DRef = useRef<any>(null);

  const graphData = useMemo(() => {
    return buildGraph(resources);
  }, [resources]);

  useEffect(() => {
    if (graphData.nodes.length === 0) {
      return;
    }

    const timer = window.setTimeout(() => {
      if (viewMode === "2d") {
        graph2DRef.current?.zoomToFit?.(420, 66);
      } else {
        graph3DRef.current?.zoomToFit?.(420, 66);
      }
    }, 120);

    return () => {
      window.clearTimeout(timer);
    };
  }, [graphData, viewMode]);

  useEffect(() => {
    if (!selectedResourceId) {
      return;
    }

    const selectedNode = graphData.nodes.find((node) => node.resourceId === selectedResourceId);
    if (!selectedNode) {
      return;
    }

    const x = toFinite(selectedNode.x);
    const y = toFinite(selectedNode.y);
    const z = toFinite(selectedNode.z);

    if (viewMode === "2d") {
      graph2DRef.current?.centerAt?.(x, y, 480);
      graph2DRef.current?.zoom?.(2.6, 480);
      return;
    }

    const distance = 170;
    const norm = Math.hypot(x, y, z) || 1;
    const ratio = 1 + distance / norm;

    graph3DRef.current?.cameraPosition?.({ x: x * ratio, y: y * ratio, z: z * ratio }, { x, y, z }, 900);
  }, [selectedResourceId, graphData, viewMode]);

  const handleNodeClick = (nodeObject: unknown) => {
    const node = nodeObject as GraphNode;
    if (node.kind === "resource" && node.resourceId) {
      selectResource(node.resourceId);
      return;
    }

    if (node.kind === "category" && node.categoryName) {
      setCategoryFilter(node.categoryName.toLowerCase());
    }
  };

  const nodeLabel = (nodeObject: unknown) => {
    const node = nodeObject as GraphNode;
    if (node.kind === "resource") {
      return `${node.label} (${node.categoryName || "Unsorted"})`;
    }
    return `${node.label} category`;
  };

  const nodeColor = (nodeObject: unknown) => {
    const node = nodeObject as GraphNode;
    if (node.resourceId && node.resourceId === selectedResourceId) {
      return "#c5ff4d";
    }
    return node.color;
  };

  const drawNodeLabel = (nodeObject: unknown, context: CanvasRenderingContext2D, scale: number) => {
    const node = nodeObject as GraphNode;
    const x = toFinite(node.x);
    const y = toFinite(node.y);
    const fontSize = Math.max(8, node.kind === "category" ? 12 / scale : 10 / scale);

    context.font = `${fontSize}px \"IBM Plex Mono\", monospace`;
    context.textAlign = "left";
    context.textBaseline = "middle";
    context.fillStyle = "rgba(230, 244, 255, 0.85)";
    context.fillText(node.label.slice(0, 24), x + 7, y);
  };

  const totalCategories = useMemo(() => {
    return graphData.nodes.filter((node) => node.kind === "category").length;
  }, [graphData]);

  const graphLoading = <div className="graph-loading">Loading graph renderer...</div>;

  return (
    <section className="graph-canvas panel">
      <div className="panel-heading">
        <h2>Knowledge Graph Surface</h2>
        <p>Live force graph with resource-category relationships and selection focus.</p>
      </div>

      <div className="graph-meta">
        <span>{resources.length} resource nodes</span>
        <span>{totalCategories} category hubs</span>
        <span>{graphData.links.length} links</span>
      </div>

      <div className="graph-field graph-stage">
        {graphData.nodes.length === 0 ? <p className="graph-empty">No nodes yet. Add your first resource on the right.</p> : null}

        {graphData.nodes.length > 0 && viewMode === "2d" ? (
          <Suspense fallback={graphLoading}>
            <ForceGraph2D
              ref={graph2DRef}
              graphData={graphData}
              backgroundColor="rgba(0,0,0,0)"
              cooldownTicks={120}
              d3VelocityDecay={0.2}
              linkColor={(linkObject) => (linkObject as GraphLink).color}
              linkDirectionalParticles={1}
              linkDirectionalParticleWidth={(linkObject) => (linkObject as GraphLink).strength}
              linkWidth={(linkObject) => (linkObject as GraphLink).strength}
              nodeAutoColorBy="kind"
              nodeCanvasObject={drawNodeLabel}
              nodeCanvasObjectMode={() => "after"}
              nodeColor={nodeColor}
              nodeLabel={nodeLabel}
              nodeRelSize={5}
              onNodeClick={handleNodeClick}
            />
          </Suspense>
        ) : null}

        {graphData.nodes.length > 0 && viewMode === "3d" ? (
          <Suspense fallback={graphLoading}>
            <ForceGraph3D
              ref={graph3DRef}
              graphData={graphData}
              backgroundColor="rgba(0,0,0,0)"
              cooldownTicks={160}
              d3VelocityDecay={0.2}
              linkColor={(linkObject) => (linkObject as GraphLink).color}
              linkOpacity={0.34}
              linkWidth={(linkObject) => (linkObject as GraphLink).strength}
              nodeColor={nodeColor}
              nodeLabel={nodeLabel}
              nodeResolution={18}
              nodeVal={(nodeObject) => (nodeObject as GraphNode).val}
              onNodeClick={handleNodeClick}
            />
          </Suspense>
        ) : null}
      </div>
    </section>
  );
}
