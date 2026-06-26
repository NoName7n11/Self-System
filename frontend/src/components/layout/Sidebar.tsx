import { useMemo } from "react";

import {
  IcoChevL, IcoChevR, IcoChat, IcoTasks, IcoLibrary, IcoGear,
  IcoSearch, IcoTrend, IcoLogo,
} from "../icons";
import { useLayoutStore } from "../../stores/useLayoutStore";
import { useResourceStore } from "../../stores/useResourceStore";
import type { DockTab, ResourceItem, ResourceType } from "../../types";

// ── type color lookup ────────────────────────────────────────────────────────
const TYPE_COLOR: Record<ResourceType, string> = {
  pdf:   "#F67373",
  link:  "#48C78E",
  note:  "#5B9CF6",
  doc:   "#9B59F6",
  image: "#F6739B",
};

// ── pure helpers (kept exported so existing tests can import them) ────────────
export function deriveFavorites(resources: ResourceItem[], limit = 3): Array<[string, number]> {
  const counter = new Map<string, number>();
  for (const item of resources) {
    const key = item.categoryName.trim() || "Unsorted";
    counter.set(key, (counter.get(key) ?? 0) + 1);
  }
  return [...counter.entries()]
    .sort((a, b) => b[1] - a[1])
    .slice(0, limit);
}

export function deriveRecents(resources: ResourceItem[], limit = 5): ResourceItem[] {
  return [...resources]
    .sort((a, b) => b.createdAt.localeCompare(a.createdAt))
    .slice(0, limit);
}

// ── nav config ───────────────────────────────────────────────────────────────
const NAV_ITEMS: Array<{ tab: DockTab; label: string; Icon: () => React.ReactElement }> = [
  { tab: "chat",    label: "CHAT",    Icon: IcoChat    },
  { tab: "tasks",   label: "TASKS",   Icon: IcoTasks },
  { tab: "library", label: "LIBRARY", Icon: IcoLibrary },
];

export default function Sidebar() {
  const leftCollapsed  = useLayoutStore((s) => s.leftCollapsed);
  const toggleLeft     = useLayoutStore((s) => s.toggleLeft);
  const openDockTab    = useLayoutStore((s) => s.openDockTab);
  const setRightOpen   = useLayoutStore((s) => s.setRightOpen);
  const resources      = useResourceStore((s) => s.resources);
  const selectResource = useResourceStore((s) => s.selectResource);
  const query          = useResourceStore((s) => s.filters.query);
  const setQuery       = useResourceStore((s) => s.setQuery);

  const recents = useMemo(() => deriveRecents(resources), [resources]);

  return (
    <aside className={`left-rail${leftCollapsed ? " is-collapsed" : ""}`}>
      {/* ── Header ── */}
      <div className="rail-header">
        <div className="logo-chip">
          <IcoLogo />
        </div>

        {!leftCollapsed && (
          <>
            <div className="rail-wordmark">
              <div className="rail-wordmark-name">Self Systems</div>
              <div className="rail-wordmark-sub">LOCAL · v0.1.0</div>
            </div>
            <button className="rail-collapse-btn" onClick={toggleLeft} type="button" aria-label="Collapse rail">
              <IcoChevL />
            </button>
          </>
        )}
      </div>

      {/* ── Search (expanded only) ── */}
      {!leftCollapsed && (
        <div className="rail-search">
          <IcoSearch />
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="SEARCH RESOURCES, TAGS…"
            aria-label="Search resources"
          />
          <span className="rail-search-hint">/</span>
        </div>
      )}

      {/* ── Primary Nav ── */}
      <nav className="rail-nav">
        {NAV_ITEMS.map(({ tab, label, Icon }) => (
          <button
            key={tab}
            className="nav-row"
            onClick={() => openDockTab(tab)}
            type="button"
          >
            <Icon />
            {!leftCollapsed && <span className="nav-row-label">{label}</span>}
            {!leftCollapsed && (
              <span
                className="nav-affordance"
                onClick={(e) => { e.stopPropagation(); openDockTab(tab); }}
                role="button"
                aria-label={`Open ${label} in dock`}
              >
                <IcoTrend />
              </span>
            )}
          </button>
        ))}
      </nav>

      {/* ── Recent (expanded only) ── */}
      {!leftCollapsed && recents.length > 0 && (
        <section className="rail-section">
          <div className="rail-section-label">Recent</div>
          {recents.map((item) => {
            const label = item.title || item.host || item.url || "Resource";
            const color = TYPE_COLOR[item.type ?? "link"] ?? "#9A9AA0";
            return (
              <button
                key={item.id}
                className="rail-recent-row"
                onClick={() => { selectResource(item.id); setRightOpen(true); }}
                type="button"
              >
                <span className="rail-type-swatch" style={{ background: color }} />
                <span className="rail-recent-title">{label}</span>
                <span className="rail-recent-type">{(item.type ?? "link").toUpperCase()}</span>
              </button>
            );
          })}
        </section>
      )}

      {/* ── Footer ── */}
      <div className="rail-footer">
        {leftCollapsed ? (
          <button className="rail-collapse-btn" onClick={toggleLeft} type="button" aria-label="Expand rail">
            <IcoChevR />
          </button>
        ) : (
          <>
            <div className="rail-avatar">N</div>
            <div className="rail-user">
              <div className="rail-user-name">noname</div>
              <div className="rail-user-sub">local · single user</div>
            </div>
            <button className="rail-gear-btn" type="button" aria-label="Settings">
              <IcoGear />
            </button>
          </>
        )}
      </div>
    </aside>
  );
}
