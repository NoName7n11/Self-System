import { useMemo } from "react";

import { useResourceStore } from "../../stores/useResourceStore";
import type { ResourceItem } from "../../types";

interface ResourceListProps {
  resources: ResourceItem[];
}

function formatDate(isoTimestamp: string): string {
  if (!isoTimestamp) {
    return "Unknown";
  }

  const value = new Date(isoTimestamp);
  if (Number.isNaN(value.getTime())) {
    return "Unknown";
  }

  return value.toLocaleDateString();
}

export default function ResourceList({ resources }: ResourceListProps) {
  const isLoading = useResourceStore((state) => state.isLoading);
  const selectedResourceId = useResourceStore((state) => state.selectedResourceId);
  const selectResource = useResourceStore((state) => state.selectResource);

  const overrideCount = useMemo(() => resources.filter((item) => item.userOverride).length, [resources]);

  return (
    <section className="resource-list panel">
      <div className="panel-heading">
        <h2>Resource Ledger</h2>
        <p>
          {resources.length} visible, {overrideCount} override-tagged
        </p>
      </div>

      {isLoading && resources.length === 0 ? <p className="muted-copy">Loading resources...</p> : null}

      {!isLoading && resources.length === 0 ? (
        <p className="muted-copy">No resources match the current filters.</p>
      ) : null}

      <div className="resource-list-scroll">
        {resources.map((resource) => (
          <button
            key={resource.id}
            className={`resource-row ${selectedResourceId === resource.id ? "is-selected" : ""}`}
            onClick={() => selectResource(resource.id)}
            type="button"
          >
            <div className="resource-row-main">
              <h3>{resource.title || resource.url}</h3>
              <p>{resource.summary || "No summary yet."}</p>
            </div>
            <div className="resource-row-meta">
              <span className="resource-chip">{resource.categoryName || "Unsorted"}</span>
              <span>{formatDate(resource.updatedAt || resource.createdAt)}</span>
            </div>
          </button>
        ))}
      </div>
    </section>
  );
}
