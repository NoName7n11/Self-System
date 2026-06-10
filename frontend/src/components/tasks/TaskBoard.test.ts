import { describe, expect, it } from "vitest";

import { formatDateTime } from "./TaskBoard";

describe("TaskBoard formatDateTime", () => {
  it("returns a placeholder when empty", () => {
    expect(formatDateTime("")).toBe("No time set");
    expect(formatDateTime("   ")).toBe("No time set");
  });

  it("returns invalid marker when parsing fails", () => {
    expect(formatDateTime("not-a-date")).toBe("Invalid time");
  });

  it("formats valid timestamps using the current locale", () => {
    const timestamp = "2026-04-20T10:30:00Z";
    const expected = new Date(timestamp).toLocaleString();

    expect(formatDateTime(timestamp)).toBe(expected);
  });
});
