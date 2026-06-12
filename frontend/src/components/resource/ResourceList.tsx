import { useMemo, useRef, useState } from "react";

import { useResourceStore } from "../../stores/useResourceStore";
import type { ResourceItem } from "../../types";

interface ResourceListProps {
	resources: ResourceItem[];
}

export function formatResourceDate(isoTimestamp: string): string {
	if (!isoTimestamp) {
		return "Unknown";
	}

	const value = new Date(isoTimestamp);
	if (Number.isNaN(value.getTime())) {
		return "Unknown";
	}

	return value.toLocaleDateString();
}

export function getResourceListStatusMessage(isLoading: boolean, resourceCount: number): string | null {
	if (isLoading && resourceCount === 0) {
		return "Loading resources...";
	}

	if (!isLoading && resourceCount === 0) {
		return "No resources match the current filters.";
	}

	return null;
}

// Below this count, render every row normally (cheap, and keeps existing
// tests that assert on full DOM output unaffected). Above it, switch to
// windowed rendering so large libraries don't bog down scrolling (Change 13 WS4).
export const RESOURCE_LIST_VIRTUALIZE_THRESHOLD = 200;

// Estimated row height in px, used to size the virtual scroll window.
export const RESOURCE_ROW_HEIGHT_PX = 92;

// Extra rows rendered above/below the visible viewport to avoid blank flashes
// while scrolling.
export const RESOURCE_LIST_OVERSCAN = 6;

export interface VirtualRange {
	startIndex: number;
	endIndex: number; // exclusive
	paddingTop: number;
	paddingBottom: number;
}

export function getVirtualRange(
	resourceCount: number,
	scrollTop: number,
	viewportHeight: number,
	rowHeight: number = RESOURCE_ROW_HEIGHT_PX,
	overscan: number = RESOURCE_LIST_OVERSCAN,
): VirtualRange {
	if (resourceCount === 0 || rowHeight <= 0 || viewportHeight <= 0) {
		return { startIndex: 0, endIndex: resourceCount, paddingTop: 0, paddingBottom: 0 };
	}

	const visibleCount = Math.ceil(viewportHeight / rowHeight);
	const firstVisible = Math.floor(scrollTop / rowHeight);

	const startIndex = Math.max(0, firstVisible - overscan);
	const endIndex = Math.min(resourceCount, firstVisible + visibleCount + overscan);

	return {
		startIndex,
		endIndex,
		paddingTop: startIndex * rowHeight,
		paddingBottom: (resourceCount - endIndex) * rowHeight,
	};
}

export default function ResourceList({ resources }: ResourceListProps) {
	const isLoading = useResourceStore((state) => state.isLoading);
	const selectedResourceId = useResourceStore((state) => state.selectedResourceId);
	const selectResource = useResourceStore((state) => state.selectResource);

	const scrollRef = useRef<HTMLDivElement>(null);
	const [scrollTop, setScrollTop] = useState(0);
	const [viewportHeight, setViewportHeight] = useState(0);

	const overrideCount = useMemo(() => resources.filter((item) => item.userOverride).length, [resources]);
	const statusMessage = getResourceListStatusMessage(isLoading, resources.length);

	const virtualize = resources.length > RESOURCE_LIST_VIRTUALIZE_THRESHOLD;
	const range = useMemo(() => {
		if (!virtualize) {
			return { startIndex: 0, endIndex: resources.length, paddingTop: 0, paddingBottom: 0 };
		}
		return getVirtualRange(resources.length, scrollTop, viewportHeight);
	}, [virtualize, resources.length, scrollTop, viewportHeight]);

	const visibleResources = resources.slice(range.startIndex, range.endIndex);

	const handleScroll = (event: React.UIEvent<HTMLDivElement>) => {
		if (!virtualize) {
			return;
		}
		const target = event.currentTarget;
		setScrollTop(target.scrollTop);
		if (target.clientHeight !== viewportHeight) {
			setViewportHeight(target.clientHeight);
		}
	};

	return (
		<section className="resource-list panel">
			<div className="panel-heading">
				<h2>Resource Ledger</h2>
				<p>
					{resources.length} visible, {overrideCount} override-tagged
				</p>
			</div>

			{statusMessage ? <p className="muted-copy">{statusMessage}</p> : null}

			<div className="resource-list-scroll" ref={scrollRef} onScroll={handleScroll}>
				{range.paddingTop > 0 ? <div style={{ height: range.paddingTop }} aria-hidden="true" /> : null}
				{visibleResources.map((resource) => (
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
							<span>{formatResourceDate(resource.updatedAt || resource.createdAt)}</span>
						</div>
					</button>
				))}
				{range.paddingBottom > 0 ? <div style={{ height: range.paddingBottom }} aria-hidden="true" /> : null}
			</div>
		</section>
	);
}
