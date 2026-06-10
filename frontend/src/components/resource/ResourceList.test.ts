import { describe, expect, it } from "vitest";

import { formatResourceDate, getResourceListStatusMessage } from "./ResourceList";

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
