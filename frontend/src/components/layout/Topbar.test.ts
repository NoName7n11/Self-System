import { describe, expect, it } from "vitest";

import type { SyncStatus } from "../../types";
import { getRuntimeClass, getRuntimeTitle, getSyncStatusLabel } from "./Topbar";

describe("Topbar helpers", () => {
  it("builds sync status labels with polling and refresh suffixes", () => {
    expect(getSyncStatusLabel("connected", 0, false, false)).toBe("Connected");
    expect(getSyncStatusLabel("offline", 0, false, true)).toBe("Offline • Refreshing");
    expect(getSyncStatusLabel("reconnecting", 2, true, true)).toBe("Reconnecting (2) • Polling • Refreshing");
  });

  it("maps sync status to runtime classes", () => {
    const cases: Array<[SyncStatus, string]> = [
      ["connected", "is-connected"],
      ["connecting", "is-reconnecting"],
      ["reconnecting", "is-reconnecting"],
      ["offline", "is-offline"],
      ["idle", "is-idle"],
    ];

    for (const [status, expected] of cases) {
      expect(getRuntimeClass(status)).toBe(expected);
    }
  });

  it("builds runtime titles from errors and last events", () => {
    expect(getRuntimeTitle("  socket error ", null, false)).toBe("socket error");
    expect(getRuntimeTitle("", " sync.todo.updated ", false)).toBe("Last sync event: sync.todo.updated");
    expect(getRuntimeTitle(null, null, false)).toBe("Sync status indicator");
    expect(getRuntimeTitle("sync down", null, true)).toBe("sync down (fallback polling active)");
  });
});
