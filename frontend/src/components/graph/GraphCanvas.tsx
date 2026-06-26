import { useCallback, useEffect, useMemo, useRef } from "react";

import { useLayoutStore } from "../../stores/useLayoutStore";
import { useResourceStore } from "../../stores/useResourceStore";
import type { ResourceItem, ResourceType } from "../../types";

// ─── Types ───────────────────────────────────────────────────────────────────

interface SimNode {
  id: string;
  kind: "cat" | "res";
  mass: number;
  r: number;
  color: string;
  label: string;
  catId: string;
  resourceId?: string;
  type?: ResourceType;
  x: number;
  y: number;
  vx: number;
  vy: number;
  fixed: boolean;
}

interface SimLink {
  a: SimNode;
  b: SimNode;
  len: number;
  k: number;
  strong: boolean;
  color: string;
}

interface Transform { scale: number; tx: number; ty: number; }

// ─── Color helpers ────────────────────────────────────────────────────────────

const CAT_COLORS: Record<string, string> = {
  research: "#5B9CF6",
  ai:       "#A98BF5",
  finance:  "#48C78E",
  people:   "#E5B567",
  sources:  "#56B6C2",
  archive:  "#E06C75",
};

const TYPE_COLORS: Record<ResourceType, string> = {
  pdf:   "#F67373",
  link:  "#48C78E",
  note:  "#5B9CF6",
  doc:   "#9B59F6",
  image: "#F6739B",
};

function catColor(name: string): string {
  const key = name.trim().toLowerCase();
  if (CAT_COLORS[key]) return CAT_COLORS[key];
  // deterministic hash fallback
  let h = 0;
  for (let i = 0; i < key.length; i++) h = (h * 31 + key.charCodeAt(i)) >>> 0;
  return `hsl(${175 + (h % 120)} 78% 63%)`;
}

// ─── Sim builder ─────────────────────────────────────────────────────────────

export function buildSimData(resources: ResourceItem[]): { nodes: SimNode[]; links: SimLink[] } {
  const catMap = new Map<string, SimNode>();
  const resNodes: SimNode[] = [];
  const links: SimLink[] = [];

  for (const r of resources) {
    const catName = r.categoryName.trim() || "Unsorted";
    const catKey = catName.toLowerCase();

    if (!catMap.has(catKey)) {
      const catNode: SimNode = {
        id: `cat:${catKey}`,
        kind: "cat",
        mass: 4,
        r: 13,
        color: catColor(catName),
        label: catName,
        catId: `cat:${catKey}`,
        x: 0, y: 0, vx: 0, vy: 0, fixed: false,
      };
      catMap.set(catKey, catNode);
    }

    const hub = catMap.get(catKey)!;
    const resColor = TYPE_COLORS[r.type ?? "link"] ?? "#5B9CF6";
    const resNode: SimNode = {
      id: r.id,
      kind: "res",
      mass: 1,
      r: 5,
      color: resColor,
      label: r.title.trim() || r.host.trim() || r.url.trim() || "Resource",
      catId: hub.id,
      resourceId: r.id,
      type: r.type,
      x: 0, y: 0, vx: 0, vy: 0, fixed: false,
    };
    resNodes.push(resNode);

    links.push({
      a: resNode, b: hub,
      len: 90, k: 0.04, strong: true, color: hub.color,
    });
  }

  const catNodes = [...catMap.values()];

  // seed hubs on circle
  const hubR = 170;
  catNodes.forEach((hub, i) => {
    const angle = (i / catNodes.length) * Math.PI * 2;
    hub.x = Math.cos(angle) * hubR;
    hub.y = Math.sin(angle) * hubR;
  });

  // seed resources near hub
  resNodes.forEach((node) => {
    const hub = catNodes.find((c) => c.id === node.catId);
    if (hub) {
      node.x = hub.x + (Math.random() - 0.5) * 110;
      node.y = hub.y + (Math.random() - 0.5) * 110;
    }
  });

  // weak hub-hub links
  for (let i = 0; i < catNodes.length; i++) {
    for (let j = i + 1; j < catNodes.length; j++) {
      links.push({
        a: catNodes[i], b: catNodes[j],
        len: 230, k: 0.02, strong: false, color: "#3A3A42",
      });
    }
  }

  return { nodes: [...catNodes, ...resNodes], links };
}

// ─── Physics tick ─────────────────────────────────────────────────────────────

function tick(nodes: SimNode[], links: SimLink[]) {
  // repulsion (all pairs)
  for (let i = 0; i < nodes.length; i++) {
    for (let j = i + 1; j < nodes.length; j++) {
      const a = nodes[i];
      const b = nodes[j];
      const dx = a.x - b.x;
      const dy = a.y - b.y;
      const d2 = Math.max(dx * dx + dy * dy, 0.01);
      const d = Math.sqrt(d2);
      const charge = a.kind === "cat" || b.kind === "cat" ? 26000 : 9000;
      const f = charge / d2;
      const fx = (f * dx) / d;
      const fy = (f * dy) / d;
      if (!a.fixed) { a.vx += fx / a.mass; a.vy += fy / a.mass; }
      if (!b.fixed) { b.vx -= fx / b.mass; b.vy -= fy / b.mass; }
    }
  }

  // springs
  for (const lk of links) {
    const dx = lk.b.x - lk.a.x;
    const dy = lk.b.y - lk.a.y;
    const d = Math.max(Math.sqrt(dx * dx + dy * dy), 0.01);
    const f = (d - lk.len) * lk.k;
    const fx = (f * dx) / d;
    const fy = (f * dy) / d;
    if (!lk.a.fixed) { lk.a.vx += fx; lk.a.vy += fy; }
    if (!lk.b.fixed) { lk.b.vx -= fx; lk.b.vy -= fy; }
  }

  // gravity + integrate
  for (const n of nodes) {
    if (n.fixed) continue;
    const g = n.kind === "cat" ? 0.004 : 0.012;
    n.vx += -n.x * g;
    n.vy += -n.y * g;
    n.vx *= 0.9;
    n.vy *= 0.9;
    const speed = Math.sqrt(n.vx * n.vx + n.vy * n.vy);
    if (speed > 14) { n.vx = (n.vx / speed) * 14; n.vy = (n.vy / speed) * 14; }
    n.x += n.vx;
    n.y += n.vy;
  }
}

// ─── Fit ─────────────────────────────────────────────────────────────────────

function fitNodes(width: number, height: number, nodes: SimNode[], tr: Transform) {
  if (nodes.length === 0) return;
  let minX = Infinity, maxX = -Infinity, minY = Infinity, maxY = -Infinity;
  for (const n of nodes) {
    minX = Math.min(minX, n.x - n.r);
    maxX = Math.max(maxX, n.x + n.r);
    minY = Math.min(minY, n.y - n.r);
    maxY = Math.max(maxY, n.y + n.r);
  }
  const pad = 70;
  const sx = (width  - pad * 2) / Math.max(maxX - minX, 1);
  const sy = (height - pad * 2) / Math.max(maxY - minY, 1);
  tr.scale = Math.min(1.8, Math.max(0.3, Math.min(sx, sy)));
  tr.tx = (width  - (maxX + minX) * tr.scale) / 2;
  tr.ty = (height - (maxY + minY) * tr.scale) / 2;
}

// ─── Renderer ─────────────────────────────────────────────────────────────────

function draw(
  ctx: CanvasRenderingContext2D,
  w: number,
  h: number,
  nodes: SimNode[],
  links: SimLink[],
  tr: Transform,
  selectedId: string | null,
  selectedCat: string | null,
  query: string,
) {
  ctx.clearRect(0, 0, w, h);
  ctx.save();
  ctx.translate(tr.tx, tr.ty);
  ctx.scale(tr.scale, tr.scale);

  const hasFilter = query.trim() !== "" || selectedCat !== null;
  const q = query.trim().toLowerCase();

  const matchesFilter = (n: SimNode): boolean => {
    if (!hasFilter) return true;
    if (selectedCat && n.catId !== selectedCat && n.id !== selectedCat) return false;
    if (q && !n.label.toLowerCase().includes(q)) return false;
    return true;
  };

  // edges
  for (const lk of links) {
    const dimA = hasFilter && !matchesFilter(lk.a);
    const dimB = hasFilter && !matchesFilter(lk.b);
    const alpha = dimA && dimB ? 0.03 : lk.strong ? 0.32 : 0.6;
    ctx.globalAlpha = alpha;
    ctx.strokeStyle = lk.color;
    ctx.lineWidth = lk.strong ? 1.5 / tr.scale : 1 / tr.scale;
    if (!lk.strong) {
      ctx.setLineDash([2 / tr.scale, 3 / tr.scale]);
    } else {
      ctx.setLineDash([]);
    }
    ctx.beginPath();
    ctx.moveTo(lk.a.x, lk.a.y);
    ctx.lineTo(lk.b.x, lk.b.y);
    ctx.stroke();
  }
  ctx.setLineDash([]);
  ctx.globalAlpha = 1;

  // nodes
  for (const n of nodes) {
    const dim = hasFilter && !matchesFilter(n);
    const alpha = dim ? 0.22 : 1;
    ctx.globalAlpha = alpha;

    if (n.kind === "cat") {
      const sz = Math.max(10, n.r * tr.scale) / tr.scale;
      ctx.fillStyle = n.color;
      ctx.fillRect(n.x - sz / 2, n.y - sz / 2, sz, sz);
      const inner = sz * 0.45;
      ctx.fillStyle = "#0B0B0D";
      ctx.fillRect(n.x - inner / 2, n.y - inner / 2, inner, inner);
      ctx.fillStyle = "#E9E9EC";
      ctx.font = `${11 / tr.scale}px 'JetBrains Mono', monospace`;
      ctx.textAlign = "center";
      ctx.fillText(n.label, n.x, n.y + sz / 2 + 14 / tr.scale);
    } else {
      const isSelected = n.resourceId === selectedId;
      const sz = Math.max(4, n.r * tr.scale * 0.95) / tr.scale;

      if (isSelected) {
        ctx.fillStyle = "rgba(240,112,60,0.12)";
        ctx.fillRect(n.x - sz / 2, n.y - sz / 2, sz, sz);
        ctx.strokeStyle = "#F0703C";
        ctx.lineWidth = 2 / tr.scale;
        ctx.strokeRect(n.x - sz / 2, n.y - sz / 2, sz, sz);
      } else {
        ctx.fillStyle = n.color;
        ctx.fillRect(n.x - sz / 2, n.y - sz / 2, sz, sz);
      }

      if (tr.scale > 1.35) {
        ctx.fillStyle = "#9A9AA0";
        ctx.font = `${9 / tr.scale}px 'JetBrains Mono', monospace`;
        ctx.textAlign = "center";
        ctx.fillText(n.label.slice(0, 20), n.x, n.y + sz / 2 + 10 / tr.scale);
      }
    }
  }

  ctx.globalAlpha = 1;
  ctx.restore();
}

// ─── Component ────────────────────────────────────────────────────────────────

interface Props { resources: ResourceItem[]; }

export default function GraphCanvas({ resources }: Props) {
  const canvasRef   = useRef<HTMLCanvasElement>(null);
  const nodesRef    = useRef<SimNode[]>([]);
  const linksRef    = useRef<SimLink[]>([]);
  const trRef       = useRef<Transform>({ scale: 1, tx: 0, ty: 0 });
  const rafRef      = useRef<number>(0);
  const warmRef     = useRef(false);

  // interaction refs
  const dragNode    = useRef<SimNode | null>(null);
  const dragStartX  = useRef(0);
  const dragStartY  = useRef(0);
  const hasDragged  = useRef(false);
  const isPanning   = useRef(false);
  const panStartX   = useRef(0);
  const panStartY   = useRef(0);

  const view        = useLayoutStore((s) => s.view);
  const selectedCat = useLayoutStore((s) => s.selectedCat);
  const setSelectedCat = useLayoutStore((s) => s.setSelectedCat);
  const setRightOpen   = useLayoutStore((s) => s.setRightOpen);
  const query       = useResourceStore((s) => s.filters.query);
  const selectedId  = useResourceStore((s) => s.selectedResourceId);
  const selectResource = useResourceStore((s) => s.selectResource);

  // store latest values in refs so canvas callbacks never go stale
  const queryRef    = useRef(query);
  const selIdRef    = useRef(selectedId);
  const selCatRef   = useRef(selectedCat);
  useEffect(() => { queryRef.current = query; }, [query]);
  useEffect(() => { selIdRef.current = selectedId; }, [selectedId]);
  useEffect(() => { selCatRef.current = selectedCat; }, [selectedCat]);

  // rebuild sim when resource list changes
  useEffect(() => {
    const { nodes, links } = buildSimData(resources);
    nodesRef.current = nodes;
    linksRef.current = links;
    warmRef.current = false;
  }, [resources]);

  // rAF loop
  const loop = useCallback(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;

    const nodes = nodesRef.current;
    const links = linksRef.current;
    const tr    = trRef.current;
    const w     = canvas.width;
    const h     = canvas.height;

    // warm-up: 260 silent steps on first run
    if (!warmRef.current && nodes.length > 0) {
      for (let i = 0; i < 260; i++) tick(nodes, links);
      fitNodes(w, h, nodes, tr);
      warmRef.current = true;

      // schedule re-fits to settle nicely
      setTimeout(() => fitNodes(w, h, nodesRef.current, trRef.current), 60);
      setTimeout(() => fitNodes(w, h, nodesRef.current, trRef.current), 360);
      setTimeout(() => fitNodes(w, h, nodesRef.current, trRef.current), 900);
    }

    tick(nodes, links);
    draw(ctx, w, h, nodes, links, tr, selIdRef.current, selCatRef.current, queryRef.current);

    rafRef.current = requestAnimationFrame(loop);
  }, []);

  useEffect(() => {
    rafRef.current = requestAnimationFrame(loop);
    return () => cancelAnimationFrame(rafRef.current);
  }, [loop]);

  // canvas resize
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ro = new ResizeObserver(() => {
      canvas.width  = canvas.offsetWidth;
      canvas.height = canvas.offsetHeight;
      fitNodes(canvas.width, canvas.height, nodesRef.current, trRef.current);
    });
    ro.observe(canvas);
    return () => ro.disconnect();
  }, []);

  // interaction handlers
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const tr = trRef.current;

    const hitTest = (ex: number, ey: number): SimNode | null => {
      const wx = (ex - tr.tx) / tr.scale;
      const wy = (ey - tr.ty) / tr.scale;
      const slop = 6 / tr.scale;
      for (let i = nodesRef.current.length - 1; i >= 0; i--) {
        const n = nodesRef.current[i];
        if (Math.abs(wx - n.x) <= n.r + slop && Math.abs(wy - n.y) <= n.r + slop) return n;
      }
      return null;
    };

    const onDown = (e: MouseEvent) => {
      const hit = hitTest(e.offsetX, e.offsetY);
      if (hit) {
        dragNode.current  = hit;
        hit.fixed         = true;
        dragStartX.current = e.offsetX;
        dragStartY.current = e.offsetY;
        hasDragged.current = false;
      } else {
        isPanning.current = true;
        panStartX.current = e.offsetX - tr.tx;
        panStartY.current = e.offsetY - tr.ty;
      }
    };

    const onMove = (e: MouseEvent) => {
      if (dragNode.current) {
        const dx = e.offsetX - dragStartX.current;
        const dy = e.offsetY - dragStartY.current;
        if (Math.abs(dx) > 3 || Math.abs(dy) > 3) hasDragged.current = true;
        dragNode.current.x  = (e.offsetX - tr.tx) / tr.scale;
        dragNode.current.y  = (e.offsetY - tr.ty) / tr.scale;
        dragNode.current.vx = 0;
        dragNode.current.vy = 0;
      } else if (isPanning.current) {
        tr.tx = e.offsetX - panStartX.current;
        tr.ty = e.offsetY - panStartY.current;
      }
    };

    const onUp = (_e: MouseEvent) => {
      if (dragNode.current) {
        if (!hasDragged.current) {
          const n = dragNode.current;
          if (n.kind === "res" && n.resourceId) {
            selectResource(n.resourceId);
            setRightOpen(true);
          } else if (n.kind === "cat") {
            setSelectedCat(selCatRef.current === n.id ? null : n.id);
          }
        }
        dragNode.current.fixed = false;
        dragNode.current = null;
      }
      isPanning.current = false;
    };

    const onWheel = (e: WheelEvent) => {
      e.preventDefault();
      const factor = e.deltaY < 0 ? 1.1 : 1 / 1.1;
      const wx = (e.offsetX - tr.tx) / tr.scale;
      const wy = (e.offsetY - tr.ty) / tr.scale;
      tr.scale = Math.min(2.6, Math.max(0.3, tr.scale * factor));
      tr.tx = e.offsetX - wx * tr.scale;
      tr.ty = e.offsetY - wy * tr.scale;
    };

    canvas.addEventListener("mousedown", onDown);
    canvas.addEventListener("mousemove", onMove);
    canvas.addEventListener("mouseup",   onUp);
    canvas.addEventListener("wheel",     onWheel, { passive: false });
    return () => {
      canvas.removeEventListener("mousedown", onDown);
      canvas.removeEventListener("mousemove", onMove);
      canvas.removeEventListener("mouseup",   onUp);
      canvas.removeEventListener("wheel",     onWheel);
    };
  }, [selectResource, setRightOpen, setSelectedCat]);

  // zoom control callbacks
  const doFit = () => {
    const canvas = canvasRef.current;
    if (canvas) fitNodes(canvas.width, canvas.height, nodesRef.current, trRef.current);
  };
  const do100 = () => {
    const canvas = canvasRef.current;
    if (canvas) { trRef.current.scale = 1; trRef.current.tx = canvas.width / 2; trRef.current.ty = canvas.height / 2; }
  };
  const doZoom = (delta: number) => {
    const tr = trRef.current;
    const canvas = canvasRef.current;
    if (!canvas) return;
    const cx = canvas.width / 2;
    const cy = canvas.height / 2;
    const factor = delta > 0 ? 1.1 : 1 / 1.1;
    const wx = (cx - tr.tx) / tr.scale;
    const wy = (cy - tr.ty) / tr.scale;
    tr.scale = Math.min(2.6, Math.max(0.3, tr.scale * factor));
    tr.tx = cx - wx * tr.scale;
    tr.ty = cy - wy * tr.scale;
  };

  // derive HUD values from last render (approximate)
  const nodeCount = nodesRef.current.length;
  const linkCount = linksRef.current.length;
  const zoom = Math.round(trRef.current.scale * 100);

  // map view: category → its resources
  const mapGroups = useMemo(() => {
    const groups = new Map<string, { name: string; color: string; items: ResourceItem[] }>();
    for (const r of resources) {
      const key = r.categoryId || "unsorted";
      if (!groups.has(key)) groups.set(key, { name: r.categoryName || "Unsorted", color: CAT_COLORS[key] ?? "#5B9CF6", items: [] });
      groups.get(key)!.items.push(r);
    }
    return [...groups.values()];
  }, [resources]);

  // progress view: "recently completed" = a few recent resources
  const recentlyDone = useMemo(() => resources.slice(0, 6), [resources]);

  return (
    <div className="graph-zone">
      {view === "graph" && (
        <>
          <canvas ref={canvasRef} className="graph-canvas" />

          <div className="graph-hud">
            {`NODES ${nodeCount} · EDGES ${linkCount} · ZOOM ${zoom}%`}
          </div>

          <div className="zoom-controls">
            <button className="zoom-btn" onClick={() => doZoom(-1)} type="button">−</button>
            <button className="zoom-btn zoom-fit" onClick={doFit} type="button">FIT</button>
            <button className="zoom-btn zoom-100" onClick={do100} type="button">100%</button>
            <button className="zoom-btn" onClick={() => doZoom(1)} type="button">+</button>
          </div>
        </>
      )}

      {view === "map" && (
        <div className="graph-overlay map-overlay">
          {mapGroups.map((g) => (
            <div key={g.name} className="map-group">
              <div className="map-hub" style={{ borderColor: g.color }}>
                <span className="map-hub-dot" style={{ background: g.color }} />
                <span className="map-hub-name">{g.name}</span>
                <span className="map-hub-count">{g.items.length}</span>
              </div>
              <div className="map-children">
                {g.items.map((r) => (
                  <button key={r.id} className="map-node" onClick={() => { selectResource(r.id); setRightOpen(true); }} type="button">
                    <span className="map-node-dot" style={{ background: TYPE_COLORS[r.type ?? "link"] ?? "#9A9AA0" }} />
                    <span className="map-node-label">{r.title || r.url}</span>
                  </button>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}

      {view === "progress" && (
        <div className="graph-overlay">
          <div className="graph-overlay-title">PROCESSING QUEUE</div>
          <div className="progress-empty">NO ACTIVE PROCESSING · ADD A RESOURCE FROM THE LIBRARY TAB</div>
          <div className="graph-overlay-title" style={{ marginTop: 24 }}>RECENTLY COMPLETED</div>
          {recentlyDone.map((r) => (
            <div key={r.id} className="overlay-row" onClick={() => { selectResource(r.id); setRightOpen(true); }}
              role="button" tabIndex={0} onKeyDown={(e) => { if (e.key === "Enter") { selectResource(r.id); setRightOpen(true); } }}>
              <span className="progress-check">✓</span>
              <span className={`type-badge ${r.type ?? "link"}`}>{(r.type ?? "link").toUpperCase()}</span>
              <span className="overlay-row-title">{r.title || r.url}</span>
              <span className="overlay-row-meta">{r.categoryName} · IN GRAPH</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
