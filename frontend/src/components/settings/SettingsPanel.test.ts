import { describe, expect, it } from "vitest";

import { firstNonEmpty } from "./SettingsPanel";

describe("SettingsPanel helpers", () => {
  it("returns the first non-empty trimmed value", () => {
    expect(firstNonEmpty("", "  ", "Primary", "Backup")).toBe("Primary");
  });

  it("returns N/A when all values are blank", () => {
    expect(firstNonEmpty("", "  ")).toBe("N/A");
  });
});
