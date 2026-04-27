import { useMemo } from "react";

import { useLayoutStore } from "../../stores/useLayoutStore";
import { useResourceStore } from "../../stores/useResourceStore";
import type { NavSection } from "../../types";

const navItems: Array<{ key: NavSection; label: string; short: string }> = [
  { key: "graph", label: "Graph", short: "GR" },
  { key: "search", label: "Search", short: "SR" },
  { key: "chat", label: "Chat", short: "CH" },
  { key: "tasks", label: "Tasks", short: "TK" },
  { key: "settings", label: "Settings", short: "ST" },
];

export default function Sidebar() {
  const activeSection = useLayoutStore((state) => state.activeSection);
  const setActiveSection = useLayoutStore((state) => state.setActiveSection);
  const sidebarCollapsed = useLayoutStore((state) => state.sidebarCollapsed);
  const toggleSidebar = useLayoutStore((state) => state.toggleSidebar);
  const resources = useResourceStore((state) => state.resources);

  const favorites = useMemo(() => {
    const counter = new Map<string, number>();
    for (const item of resources) {
      const key = item.categoryName.trim() || "Unsorted";
      counter.set(key, (counter.get(key) ?? 0) + 1);
    }

    return [...counter.entries()]
      .sort((left, right) => right[1] - left[1])
      .slice(0, 3);
  }, [resources]);

  const recents = useMemo(() => {
    return [...resources]
      .sort((left, right) => right.createdAt.localeCompare(left.createdAt))
      .slice(0, 4);
  }, [resources]);

  return (
    <aside className={`sidebar ${sidebarCollapsed ? "is-collapsed" : ""}`}>
      <div className="sidebar-brand">
        <div className="brand-mark">SS</div>
        {!sidebarCollapsed ? <div className="brand-title">Self Systems</div> : null}
      </div>

      <nav className="sidebar-nav">
        {navItems.map((item) => (
          <button
            key={item.key}
            className={`nav-item ${activeSection === item.key ? "is-active" : ""}`}
            onClick={() => setActiveSection(item.key)}
            type="button"
          >
            <span className="nav-short">{item.short}</span>
            {!sidebarCollapsed ? <span className="nav-label">{item.label}</span> : null}
          </button>
        ))}
      </nav>

      {!sidebarCollapsed ? (
        <>
          <section className="sidebar-block">
            <div className="sidebar-heading">Favorites</div>
            {favorites.length === 0 ? <p className="muted-copy">No categories yet.</p> : null}
            {favorites.map(([name, count]) => (
              <div className="sidebar-row" key={name}>
                <span>{name}</span>
                <span>{count}</span>
              </div>
            ))}
          </section>

          <section className="sidebar-block">
            <div className="sidebar-heading">Recent</div>
            {recents.length === 0 ? <p className="muted-copy">No resources yet.</p> : null}
            {recents.map((item) => (
              <div className="sidebar-row" key={item.id}>
                <span className="ellipsis">{item.title || item.url}</span>
              </div>
            ))}
          </section>
        </>
      ) : null}

      <button className="sidebar-toggle" onClick={toggleSidebar} type="button">
        {sidebarCollapsed ? "Expand" : "Collapse"}
      </button>
    </aside>
  );
}
