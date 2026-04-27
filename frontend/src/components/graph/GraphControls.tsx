import { useMemo } from "react";

import { useResourceStore } from "../../stores/useResourceStore";
import type { ResourceItem } from "../../types";

interface GraphControlsProps {
  resources: ResourceItem[];
}

export default function GraphControls({ resources }: GraphControlsProps) {
  const category = useResourceStore((state) => state.filters.category);
  const viewMode = useResourceStore((state) => state.filters.viewMode);
  const showOverridesOnly = useResourceStore((state) => state.filters.showOverridesOnly);
  const setCategoryFilter = useResourceStore((state) => state.setCategoryFilter);
  const setViewMode = useResourceStore((state) => state.setViewMode);
  const toggleOverridesOnly = useResourceStore((state) => state.toggleOverridesOnly);
  const setQuery = useResourceStore((state) => state.setQuery);

  const categories = useMemo(() => {
    return [...new Set(resources.map((item) => item.categoryName.trim()).filter((item) => item !== ""))].sort((left, right) =>
      left.localeCompare(right),
    );
  }, [resources]);

  return (
    <section className="graph-controls panel">
      <div className="panel-heading">
        <h2>Graph Controls</h2>
        <p>Filter and perspective settings for the knowledge map.</p>
      </div>

      <div className="control-grid">
        <label>
          Category
          <select onChange={(event) => setCategoryFilter(event.target.value)} value={category}>
            <option value="all">All categories</option>
            {categories.map((item) => (
              <option key={item} value={item.toLowerCase()}>
                {item}
              </option>
            ))}
          </select>
        </label>

        <div>
          <span className="control-label">View mode</span>
          <div className="mode-toggle">
            <button
              className={viewMode === "2d" ? "is-active" : ""}
              onClick={() => setViewMode("2d")}
              type="button"
            >
              2D
            </button>
            <button
              className={viewMode === "3d" ? "is-active" : ""}
              onClick={() => setViewMode("3d")}
              type="button"
            >
              3D
            </button>
          </div>
        </div>

        <label className="switch-line">
          <input checked={showOverridesOnly} onChange={toggleOverridesOnly} type="checkbox" />
          User overrides only
        </label>

        <button
          className="ghost-button"
          onClick={() => {
            setCategoryFilter("all");
            setQuery("");
          }}
          type="button"
        >
          Reset filters
        </button>
      </div>
    </section>
  );
}
