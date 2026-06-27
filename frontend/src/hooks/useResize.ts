import { useLayoutStore } from "../stores/useLayoutStore";

// Drag-to-resize + snap-to-collapse for the three panels.
// `collapseAt` is intentionally well below `min` so collapsing is NOT twitchy:
// Left/Right have a wide buffer (low sensitivity); Dock matches the design.
type Kind = "left" | "right" | "dock";

const LIMITS: Record<Kind, { floor: number; max: () => number; def: number; collapseAt: number }> = {
  // left/right: drag floor low + collapseAt far below default → must really shrink to collapse
  left:  { floor: 120, max: () => 360, def: 264, collapseAt: 150 },
  right: { floor: 150, max: () => 460, def: 336, collapseAt: 185 },
  // dock: keep design feel
  dock:  { floor: 90,  max: () => Math.round((window.innerHeight - 52) * 0.5), def: 264, collapseAt: 160 },
};

export function startPanelResize(kind: Kind, e: React.MouseEvent) {
  e.preventDefault();
  e.stopPropagation();
  const s = useLayoutStore.getState();
  const lim = LIMITS[kind];
  const startX = e.clientX;
  const startY = e.clientY;
  const startVal = kind === "left" ? s.leftWpx : kind === "right" ? s.rightWpx : s.dockHpx;

  s.setResizing(true);
  document.body.style.userSelect = "none";

  const onMove = (ev: MouseEvent) => {
    const st = useLayoutStore.getState();
    let next: number;
    if (kind === "left") next = startVal + (ev.clientX - startX);          // grows right
    else if (kind === "right") next = startVal - (ev.clientX - startX);    // left edge, grows left
    else next = startVal - (ev.clientY - startY);                          // top edge, grows up
    next = Math.max(lim.floor, Math.min(lim.max(), next));
    if (kind === "left") st.setLeftWpx(next);
    else if (kind === "right") st.setRightWpx(next);
    else st.setDockHpx(next);
  };

  const onUp = () => {
    window.removeEventListener("mousemove", onMove);
    window.removeEventListener("mouseup", onUp);
    document.body.style.userSelect = "";
    const st = useLayoutStore.getState();
    const cur = kind === "left" ? st.leftWpx : kind === "right" ? st.rightWpx : st.dockHpx;
    st.setResizing(false);
    // snap to collapsed only when dragged well past the comfortable minimum
    if (cur < lim.collapseAt) {
      if (kind === "left") { st.setLeftCollapsed(true); st.setLeftWpx(lim.def); }
      else if (kind === "right") { st.setRightOpen(false); st.setRightWpx(lim.def); }
      else { st.setDockOpen(false); st.setDockHpx(lim.def); }
    }
  };

  window.addEventListener("mousemove", onMove);
  window.addEventListener("mouseup", onUp);
}
