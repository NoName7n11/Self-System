import { describe, expect, it } from "vitest";

import {
  RESOURCE_LIST_VIRTUALIZE_THRESHOLD,
  RESOURCE_ROW_HEIGHT_PX,
  formatResourceDate,
  getResourceListStatusMessage,
  getVirtualRange,
} from "./ResourceList";

describe("ResourceList helpers", () => {
  it("returns Unknown when the timestamp is missing or invalid", () => {
    expect(formatResourceDate("")).toBe("Unknown");
    expect(formatResourceDate("not-a-date")).toBe("Unknown");
  });

  it("formats valid timestamps using the current locale", () => {
    const timestamp = "2026-04-21T10:05:00Z";
    const expected = new Date(timestamp).toLocaleDateString();

    expect(formatResourceDate(timestamp)).toBe(expected);
  });

  it("returns a loading message when empty and loading", () => {
    expect(getResourceListStatusMessage(true, 0)).toBe("Loading resources...");
  });

  it("returns an empty-state message when empty and not loading", () => {
    expect(getResourceListStatusMessage(false, 0)).toBe("No resources match the current filters.");
  });

  it("returns null when there are resources", () => {
    expect(getResourceListStatusMessage(true, 1)).toBeNull();
    expect(getResourceListStatusMessage(false, 1)).toBeNull();
  });
});

describe("getVirtualRange", () => {
  it("returns the full range when there are no resources or no viewport", () => {
    expect(getVirtualRange(0, 0, 600)).toEqual({ startIndex: 0, endIndex: 0, paddingTop: 0, paddingBottom: 0 });
    expect(getVirtualRange(10, 0, 0)).toEqual({ startIndex: 0, endIndex: 10, paddingTop: 0, paddingBottom: 0 });
  });

  it("windows around the scroll position with overscan", () => {
    const range = getVirtualRange(1000, RESOURCE_ROW_HEIGHT_PX * 50, 600);

    expect(range.startIndex).toBeLessThanOrEqual(50);
    expect(range.endIndex).toBeGreaterThan(50);
    expect(range.paddingTop).toBe(range.startIndex * RESOURCE_ROW_HEIGHT_PX);
    expect(range.paddingBottom).toBe((1000 - range.endIndex) * RESOURCE_ROW_HEIGHT_PX);
  });

  it("clamps the range to the resource count at the start and end of the list", () => {
    const start = getVirtualRange(1000, 0, 600);
    expect(start.startIndex).toBe(0);
    expect(start.paddingTop).toBe(0);

    const end = getVirtualRange(1000, RESOURCE_ROW_HEIGHT_PX * 1000, 600);
    expect(end.endIndex).toBe(1000);
    expect(end.paddingBottom).toBe(0);
  });

  it("covers a synthetic 10k-resource list without exceeding its bounds", () => {
    const totalResources = 10000;
    expect(totalResources).toBeGreaterThan(RESOURCE_LIST_VIRTUALIZE_THRESHOLD);

    for (const scrollTop of [0, RESOURCE_ROW_HEIGHT_PX * 5000, RESOURCE_ROW_HEIGHT_PX * 9999]) {
      const range = getVirtualRange(totalResources, scrollTop, 800);
      expect(range.startIndex).toBeGreaterThanOrEqual(0);
      expect(range.endIndex).toBeLessThanOrEqual(totalResources);
      expect(range.endIndex - range.startIndex).toBeLessThan(50);
    }
  });
});
