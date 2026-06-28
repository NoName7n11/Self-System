import { useCallback, useEffect, useMemo, useRef } from "react";

import { DEMO_CATEGORIES, DEMO_RESOURCES } from "../../lib/demoData";
import { useLayoutStore } from "../../stores/useLayoutStore";
import { useResourceStore } from "../../stores/useResourceStore";
import { useTaskStore } from "../../stores/useTaskStore";
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

function catColor(id: string): string {
  const key = id.trim().toLowerCase();
  if (CAT_COLORS[key]) return CAT_COLORS[key];
  // deterministic hash fallback for user-created categories
  let h = 0;
  for (let i = 0; i < key.length; i++) h = (h * 31 + key.charCodeAt(i)) >>> 0;
  return `hsl(${175 + (h % 120)} 78% 63%)`;
}

// Sparse weighted hub↔hub relations (matches design §10.1 CATLINKS). Only the
// edges between categories that actually have a relationship — NOT all-pairs —
// so the hub layout is organic, not a rigid symmetric polygon.
const CATLINKS: Array<[string, string]> = [
  ["research", "ai"], ["research", "sources"], ["ai", "finance"],
  ["ai", "sources"], ["people", "research"], ["finance", "people"],
  ["archive", "sources"],
];

// counter (share count → node size) + connections (res↔res edges), keyed by
// resource id. Real backend resources fall back to counter 1 / no connections.
const COUNTER_BY_ID = new Map(DEMO_RESOURCES.map((r) => [r.id, r.counter]));
const CONNS_BY_ID = new Map(DEMO_RESOURCES.map((r) => [r.id, r.connections]));

// ─── Sim builder ─────────────────────────────────────────────────────────────

export function buildSimData(resources: ResourceItem[]): { nodes: SimNode[]; links: SimLink[] } {
  const catMap = new Map<string, SimNode>();
  const resNodes: SimNode[] = [];
  const links: SimLink[] = [];

  for (const r of resources) {
    const catId = r.categoryId.trim() || "unsorted";

    if (!catMap.has(catId)) {
      catMap.set(catId, {
        id: catId,
        kind: "cat",
        mass: 4,
        r: 13,
        color: catColor(catId),
        label: r.categoryName.trim() || catId,
        catId,
        x: 0, y: 0, vx: 0, vy: 0, fixed: false,
      });
    }

    const hub = catMap.get(catId)!;
    const counter = COUNTER_BY_ID.get(r.id) ?? 1;
    const resNode: SimNode = {
      id: r.id,
      kind: "res",
      mass: 1,
      r: 5 + counter * 0.7,
      color: TYPE_COLORS[r.type ?? "link"] ?? "#5B9CF6",
      label: r.title.trim() || r.host.trim() || r.url.trim() || "Resource",
      catId: hub.id,
      resourceId: r.id,
      type: r.type,
      x: 0, y: 0, vx: 0, vy: 0, fixed: false,
    };
    resNodes.push(resNode);

    // resource → category (strong, solid, category color)
    links.push({ a: resNode, b: hub, len: 90, k: 0.04, strong: true, color: hub.color });
  }

  const catNodes = [...catMap.values()];
  const byId = new Map<string, SimNode>([...catMap, ...resNodes.map((n) => [n.id, n] as const)]);

  // seed hubs evenly on a circle, resources radially around their hub
  const hubR = 170;
  catNodes.forEach((hub, i) => {
    const a = (i / catNodes.length) * Math.PI * 2;
    hub.x = Math.cos(a) * hubR;
    hub.y = Math.sin(a) * hubR;
  });
  resNodes.forEach((node) => {
    const hub = byId.get(node.catId);
    const a = Math.random() * Math.PI * 2;
    node.x = (hub?.x ?? 0) + Math.cos(a) * 55;
    node.y = (hub?.y ?? 0) + Math.sin(a) * 55;
  });

  // weak hub↔hub links (sparse, from CATLINKS)
  for (const [a, b] of CATLINKS) {
    const na = byId.get(a), nb = byId.get(b);
    if (na && nb) links.push({ a: na, b: nb, len: 230, k: 0.02, strong: false, color: "#3A3A42" });
  }

  // weak resource↔resource links (from connections, deduped a<b)
  for (const r of resNodes) {
    for (const c of CONNS_BY_ID.get(r.id) ?? []) {
      const other = byId.get(c.to);
      if (other && r.id < c.to) links.push({ a: r, b: other, len: 78, k: 0.015, strong: false, color: "#2E2E36" });
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

function hexA(hex: string, a: number): string {
  const h = hex.replace("#", "");
  if (h.length !== 6) return hex;
  const r = parseInt(h.slice(0, 2), 16), g = parseInt(h.slice(2, 4), 16), b = parseInt(h.slice(4, 6), 16);
  return `rgba(${r},${g},${b},${a})`;
}

const ACC = "#F0703C";

// Draws in *screen space* (world→screen projection per point), like the design.
// Constant screen-px line widths and font sizes → no jitter/blur while zooming.
function draw(
  ctx: CanvasRenderingContext2D,
  dpr: number,
  cssW: number,
  cssH: number,
  nodes: SimNode[],
  links: SimLink[],
  tr: Transform,
  selectedId: string | null,
  selectedCat: string | null,
  hoverId: string | null,
  query: string,
) {
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
  ctx.clearRect(0, 0, cssW, cssH);

  const sc = tr.scale;
  const w2sx = (x: number) => x * sc + tr.tx;
  const w2sy = (y: number) => y * sc + tr.ty;

  const q = query.trim().toLowerCase();
  const ms = q
    ? new Set(
        nodes
          .filter((n) => n.label.toLowerCase().includes(q) || n.catId.toLowerCase().includes(q))
          .map((n) => n.id),
      )
    : null;
  const dimmed = (n: SimNode): boolean => {
    if (ms) return !ms.has(n.id) && !(n.kind === "res" && ms.has(n.catId));
    if (selectedCat) return n.id !== selectedCat && n.catId !== selectedCat;
    return false;
  };

  // edges
  for (const lk of links) {
    const dm = dimmed(lk.a) && dimmed(lk.b);
    ctx.beginPath();
    ctx.moveTo(w2sx(lk.a.x), w2sy(lk.a.y));
    ctx.lineTo(w2sx(lk.b.x), w2sy(lk.b.y));
    ctx.strokeStyle = lk.strong ? hexA(lk.color, dm ? 0.04 : 0.32) : hexA(lk.color, dm ? 0.03 : 0.6);
    ctx.lineWidth = lk.strong ? 1.1 : 1;
    ctx.setLineDash(lk.strong ? [] : [2, 3]);
    ctx.stroke();
  }
  ctx.setLineDash([]);

  // nodes
  ctx.textAlign = "center";
  for (const n of nodes) {
    const px = w2sx(n.x), py = w2sy(n.y);
    const dm = dimmed(n);
    const alpha = dm ? 0.22 : 1;
    const isSel = n.resourceId === selectedId || n.id === selectedId;
    const isHov = n.id === hoverId;

    if (n.kind === "cat") {
      const s = Math.max(10, n.r * sc);
      ctx.fillStyle = hexA(n.color, alpha);
      ctx.fillRect(Math.round(px - s), Math.round(py - s), Math.round(s * 2), Math.round(s * 2));
      const inner = s * 0.4;
      ctx.fillStyle = hexA("#0B0B0D", alpha);
      ctx.fillRect(Math.round(px - inner), Math.round(py - inner), Math.round(inner * 2), Math.round(inner * 2));
      ctx.font = '700 10px "JetBrains Mono", monospace';
      ctx.fillStyle = hexA("#E9E9EC", dm ? 0.3 : 0.92);
      ctx.fillText(n.label, px, py + s + 13);
    } else {
      const s = Math.max(4, n.r * sc * 0.95);
      if (isSel) {
        ctx.strokeStyle = ACC;
        ctx.lineWidth = 1.5;
        ctx.strokeRect(px - s - 4, py - s - 4, (s + 4) * 2, (s + 4) * 2);
        ctx.fillStyle = hexA(ACC, 0.12);
        ctx.fillRect(px - s - 4, py - s - 4, (s + 4) * 2, (s + 4) * 2);
      }
      ctx.fillStyle = n.color === "#5C5C66" ? hexA("#5C5C66", alpha) : hexA(n.color, isHov ? 1 : alpha);
      ctx.fillRect(Math.round(px - s), Math.round(py - s), Math.round(s * 2), Math.round(s * 2));
      if (isSel || isHov || sc > 1.35) {
        ctx.font = '500 9px "JetBrains Mono", monospace';
        ctx.fillStyle = hexA("#C9C9CF", dm ? 0.3 : isSel ? 1 : 0.7);
        ctx.fillText(n.label, px, py + s + 11);
      }
    }
  }

  // HUD (live, every frame)
  ctx.textAlign = "left";
  ctx.font = '500 9px "JetBrains Mono", monospace';
  ctx.fillStyle = "#46464E";
  ctx.fillText(`NODES ${nodes.length}  ·  EDGES ${links.length}  ·  ZOOM ${Math.round(sc * 100)}%`, 14, 20);
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
  const dprRef      = useRef(1);
  const cssWRef     = useRef(0);
  const cssHRef     = useRef(0);
  const tfInitRef   = useRef(false);
  const hoverRef    = useRef<string | null>(null);

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
    const cssW  = cssWRef.current;
    const cssH  = cssHRef.current;

    // warm-up: 260 silent steps once the canvas has a real size, then fit
    if (!warmRef.current && nodes.length > 0 && cssW > 0) {
      for (let i = 0; i < 260; i++) tick(nodes, links);
      fitNodes(cssW, cssH, nodes, tr);
      warmRef.current = true;
      // a few delayed re-fits to settle the framing (matches design)
      setTimeout(() => fitNodes(cssWRef.current, cssHRef.current, nodesRef.current, trRef.current), 60);
      setTimeout(() => fitNodes(cssWRef.current, cssHRef.current, nodesRef.current, trRef.current), 360);
      setTimeout(() => fitNodes(cssWRef.current, cssHRef.current, nodesRef.current, trRef.current), 900);
    }

    tick(nodes, links);
    draw(ctx, dprRef.current, cssW, cssH, nodes, links, tr, selIdRef.current, selCatRef.current, hoverRef.current, queryRef.current);

    rafRef.current = requestAnimationFrame(loop);
  }, []);

  useEffect(() => {
    rafRef.current = requestAnimationFrame(loop);
    return () => cancelAnimationFrame(rafRef.current);
  }, [loop]);

  // canvas resize — update backing-store size for DPR only. Do NOT refit:
  // refitting here makes the graph jump every time a panel is dragged/collapsed.
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const apply = () => {
      const dpr = window.devicePixelRatio || 1;
      const cssW = canvas.clientWidth || 800;
      const cssH = canvas.clientHeight || 500;
      dprRef.current = dpr;
      cssWRef.current = cssW;
      cssHRef.current = cssH;
      canvas.width = Math.round(cssW * dpr);
      canvas.height = Math.round(cssH * dpr);
      // center the world once, the first time we know our size
      if (!tfInitRef.current) {
        trRef.current.tx = cssW / 2;
        trRef.current.ty = cssH / 2;
        tfInitRef.current = true;
      }
    };
    apply();
    const ro = new ResizeObserver(apply);
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
      canvas.style.cursor = "grabbing";
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
      } else {
        // idle hover: highlight + cursor feedback
        const hit = hitTest(e.offsetX, e.offsetY);
        hoverRef.current = hit ? hit.id : null;
        canvas.style.cursor = hit ? "pointer" : "grab";
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
      canvas.style.cursor = "grab";
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

  // zoom control callbacks (operate in CSS px, like the design)
  const doFit = () => fitNodes(cssWRef.current, cssHRef.current, nodesRef.current, trRef.current);
  const do100 = () => {
    trRef.current.scale = 1;
    trRef.current.tx = cssWRef.current / 2;
    trRef.current.ty = cssHRef.current / 2;
  };
  const doZoom = (delta: number) => {
    const tr = trRef.current;
    const cx = cssWRef.current / 2;
    const cy = cssHRef.current / 2;
    const factor = delta > 0 ? 1.2 : 1 / 1.2;
    const wx = (cx - tr.tx) / tr.scale;
    const wy = (cy - tr.ty) / tr.scale;
    tr.scale = Math.min(2.6, Math.max(0.3, tr.scale * factor));
    tr.tx = cx - wx * tr.scale;
    tr.ty = cy - wy * tr.scale;
  };

  // progress view: "recently completed" = a few recent resources
  const recentlyDone = useMemo(() => resources.slice(0, 6), [resources]);

  // ── TASK MAP (map view): categories that have tasks → task leaves ──
  const todos = useTaskStore((s) => s.todos);
  const mapExpanded = useLayoutStore((s) => s.mapExpanded);
  const mapAllExpanded = useLayoutStore((s) => s.mapAllExpanded);
  const mapZoom = useLayoutStore((s) => s.mapZoom);
  const toggleMapNode = useLayoutStore((s) => s.toggleMapNode);
  const toggleMapAll = useLayoutStore((s) => s.toggleMapAll);
  const mapZoomIn = useLayoutStore((s) => s.mapZoomIn);
  const mapZoomOut = useLayoutStore((s) => s.mapZoomOut);
  const mapZoomReset = useLayoutStore((s) => s.mapZoomReset);
  const mapCats = useMemo(
    () => DEMO_CATEGORIES
      .map((c) => ({ ...c, tasks: todos.filter((t) => t.cat === c.id && !t.archived) }))
      .filter((c) => c.tasks.length > 0),
    [todos],
  );
  const statusColor = (s: string) => (s === "done" ? "#48C78E" : s === "in_progress" ? "#F0703C" : "#7A7A84");

  return (
    <div className="graph-zone">
      {view === "graph" && (
        <>
          <canvas ref={canvasRef} className="graph-canvas" />

          <div className="zoom-controls">
            <button className="zoom-btn" onClick={() => doZoom(-1)} type="button">−</button>
            <button className="zoom-btn zoom-fit" onClick={doFit} type="button">FIT</button>
            <button className="zoom-btn zoom-100" onClick={do100} type="button">100%</button>
            <button className="zoom-btn" onClick={() => doZoom(1)} type="button">+</button>
          </div>
        </>
      )}

      {view === "map" && (
        <div className="task-map-overlay">
          <div className="task-map" style={{ transform: `scale(${mapZoom})` }}>
            <div className="tm-root">TASKS</div>
            <div className="tm-cats">
              {mapCats.map((c) => {
                const exp = mapAllExpanded || mapExpanded[c.id];
                return (
                  <div key={c.id} className="tm-cat-row">
                    <button className="tm-node tm-cat" onClick={() => toggleMapNode(c.id)} type="button">
                      <span className="tm-dot" style={{ background: c.color }} />
                      <span className="tm-name">{c.name}</span>
                      <span className="tm-count">{c.tasks.length}</span>
                      <span className="tm-chev">{exp ? "▾" : "▸"}</span>
                    </button>
                    {exp && (
                      <div className="tm-tasks">
                        {c.tasks.map((t) => (
                          <div key={t.id} className="tm-node tm-task">
                            <span className="tm-dot" style={{ background: statusColor(t.status) }} />
                            <span className={`tm-name${t.status === "done" ? " is-done" : ""}`}>{t.title}</span>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                );
              })}
              {mapCats.length === 0 && <div className="progress-empty">NO TASKS YET</div>}
            </div>
          </div>

          <div className="map-controls">
            <button className="map-expand-btn" onClick={toggleMapAll} type="button">
              {mapAllExpanded ? "COLLAPSE ALL" : "EXPAND ALL"}
            </button>
            <div className="zoom-controls map-zoom">
              <button className="zoom-btn" onClick={mapZoomOut} type="button">−</button>
              <button className="zoom-btn zoom-fit" onClick={mapZoomReset} type="button">FIT</button>
              <button className="zoom-btn zoom-100" onClick={mapZoomReset} type="button">{Math.round(mapZoom * 100)}%</button>
              <button className="zoom-btn" onClick={mapZoomIn} type="button">+</button>
            </div>
          </div>
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
