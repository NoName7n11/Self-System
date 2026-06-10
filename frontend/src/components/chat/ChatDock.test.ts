import { describe, expect, it, vi } from "vitest";

import { presets, submitChatMessage } from "./ChatDock";

describe("ChatDock helpers", () => {
  it("defines a stable set of presets", () => {
    expect(presets).toEqual([
      "list resources",
      "list categories",
      "create category research | high-priority",
    ]);
  });

  it("does not send blank messages", async () => {
    const sendMessage = vi.fn().mockResolvedValue(undefined);
    const onClear = vi.fn();

    await submitChatMessage("   ", sendMessage, onClear);

    expect(sendMessage).not.toHaveBeenCalled();
    expect(onClear).not.toHaveBeenCalled();
  });

  it("sends trimmed input and clears on success", async () => {
    const sendMessage = vi.fn().mockResolvedValue(undefined);
    const onClear = vi.fn();

    await submitChatMessage("  list resources  ", sendMessage, onClear);

    expect(sendMessage).toHaveBeenCalledWith("list resources");
    expect(onClear).toHaveBeenCalledTimes(1);
  });
});
