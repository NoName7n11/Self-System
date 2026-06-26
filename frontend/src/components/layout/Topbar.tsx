import { IcoFilter } from "../icons";
import { useLayoutStore } from "../../stores/useLayoutStore";
import { useResourceStore } from "../../stores/useResourceStore";
import type { GraphView } from "../../types";

interface TopbarProps {
  resourceCount: number;
}

const VIEWS: Array<{ key: GraphView; label: string }> = [
  { key: "graph",    label: "GRAPH"    },
  { key: "list",     label: "LIST"     },
  { key: "timeline", label: "TIMELINE" },
];

// Kept exported for existing tests that import from Topbar.tsx
export function getSyncStatusLabel(): string { return ""; }
export function getRuntimeClass(): string { return ""; }
export function getRuntimeTitle(): string { return ""; }

export default function Topbar({ resourceCount }: TopbarProps) {
  const query    = useResourceStore((s) => s.filters.query);
  const setQuery = useResourceStore((s) => s.setQuery);
  const view     = useLayoutStore((s) => s.view);
  const setView  = useLayoutStore((s) => s.setView);

  return (
    <header className="top-bar">
      <div>
        <div className="top-bar-title-main">Knowledge Graph</div>
        <div className="top-bar-title-sub">{resourceCount} RESOURCES</div>
      </div>

      <div className="top-bar-spacer" />

      <div className="top-bar-filter">
        <IcoFilter />
        <input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="FILTER NODES…"
          aria-label="Filter graph nodes"
        />
      </div>

      <div className="view-switch" role="group" aria-label="View mode">
        {VIEWS.map(({ key, label }) => (
          <button
            key={key}
            className={`view-switch-btn${view === key ? " is-active" : ""}`}
            onClick={() => setView(key)}
            type="button"
          >
            {label}
          </button>
        ))}
      </div>
    </header>
  );
}
