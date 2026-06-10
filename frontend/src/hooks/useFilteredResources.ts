import { useMemo } from "react";

import { useResourceStore } from "../stores/useResourceStore";
import type { ResourceFilters, ResourceItem } from "../types";

export function filterResources(resources: ResourceItem[], filters: ResourceFilters) {
  const query = filters.query.trim().toLowerCase();
  const category = filters.category.trim().toLowerCase();

  return resources.filter((resource) => {
    const matchesQuery =
      query === "" ||
      resource.title.toLowerCase().includes(query) ||
      resource.summary.toLowerCase().includes(query) ||
      resource.url.toLowerCase().includes(query) ||
      resource.categoryName.toLowerCase().includes(query);

    const matchesCategory = category === "all" || resource.categoryName.toLowerCase() === category;
    const matchesOverride = !filters.showOverridesOnly || resource.userOverride;

    return matchesQuery && matchesCategory && matchesOverride;
  });
}

export function useFilteredResources() {
  const resources = useResourceStore((state) => state.resources);
  const filters = useResourceStore((state) => state.filters);

  return useMemo(() => {
    return filterResources(resources, filters);
  }, [resources, filters]);
}
