import { useMemo } from "react";

import { useResourceStore } from "../../stores/useResourceStore";
import type { ResourceItem } from "../../types";

export function getResourceFormCopy(selectedResource: ResourceItem | null) {
  if (selectedResource) {
    const label = selectedResource.title || selectedResource.url;
    return {
      heading: "Edit Resource",
      subheading: `Selected: ${label}`,
    };
  }

  return {
    heading: "Add Resource",
    subheading: "Create a new resource node.",
  };
}

export default function ResourceForm() {
  const resources = useResourceStore((state) => state.resources);
  const selectedResourceId = useResourceStore((state) => state.selectedResourceId);
  const draft = useResourceStore((state) => state.draft);
  const error = useResourceStore((state) => state.error);
  const updateDraft = useResourceStore((state) => state.updateDraft);
  const resetDraft = useResourceStore((state) => state.resetDraft);
  const addResource = useResourceStore((state) => state.addResource);
  const updateSelectedResource = useResourceStore((state) => state.updateSelectedResource);
  const deleteSelectedResource = useResourceStore((state) => state.deleteSelectedResource);

  const selectedResource = useMemo(() => {
    return resources.find((item) => item.id === selectedResourceId) ?? null;
  }, [resources, selectedResourceId]);

  const formCopy = getResourceFormCopy(selectedResource);

  return (
    <section className="resource-form panel">
      <div className="panel-heading">
        <h2>{formCopy.heading}</h2>
        <p>{formCopy.subheading}</p>
      </div>

      <label>
        URL
        <input onChange={(event) => updateDraft("url", event.target.value)} placeholder="https://..." value={draft.url} />
      </label>

      <label>
        Title
        <input onChange={(event) => updateDraft("title", event.target.value)} placeholder="Readable title" value={draft.title} />
      </label>

      <label>
        Category
        <input
          onChange={(event) => updateDraft("categoryName", event.target.value)}
          placeholder="AI, Research, Finance"
          value={draft.categoryName}
        />
      </label>

      <label>
        Summary
        <textarea
          onChange={(event) => updateDraft("summary", event.target.value)}
          placeholder="Short summary used in graph search"
          rows={4}
          value={draft.summary}
        />
      </label>

      {error ? <p className="error-copy">{error}</p> : null}

      <div className="form-actions">
        <button className="primary-button" onClick={() => void addResource()} type="button">
          Add As New
        </button>

        <button
          className="ghost-button"
          disabled={!selectedResource}
          onClick={() => void updateSelectedResource()}
          type="button"
        >
          Update Selected
        </button>

        <button
          className="ghost-button danger-button"
          disabled={!selectedResource}
          onClick={() => void deleteSelectedResource()}
          type="button"
        >
          Delete Selected
        </button>

        <button className="ghost-button" onClick={resetDraft} type="button">
          Clear
        </button>
      </div>
    </section>
  );
}
