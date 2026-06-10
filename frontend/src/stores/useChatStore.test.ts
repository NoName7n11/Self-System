import { beforeEach, describe, expect, it, vi } from "vitest";

import { sendChatCommand } from "../api/client";
import { useResourceStore } from "./useResourceStore";
import { useTaskStore } from "./useTaskStore";
import { useChatStore } from "./useChatStore";

vi.mock("../api/client", () => ({
  sendChatCommand: vi.fn(),
}));

function resetChatStore() {
  useChatStore.setState({
    messages: [
      {
        id: "assistant-seed",
        role: "assistant",
        content:
          "Command mode is active. Try: create category research | notes, resource: https://example.com | category=Research, or list resources.",
        createdAt: "2026-04-11T00:00:00.000Z",
      },
    ],
    isSending: false,
    error: null,
  });
}

describe("useChatStore", () => {
  const loadResources = vi.fn();
  const loadAll = vi.fn();

  beforeEach(() => {
    vi.restoreAllMocks();
    vi.clearAllMocks();
    resetChatStore();

    loadResources.mockResolvedValue(undefined);
    loadAll.mockResolvedValue(undefined);

    vi.spyOn(useResourceStore, "getState").mockReturnValue({
      loadResources,
    } as never);
    vi.spyOn(useTaskStore, "getState").mockReturnValue({
      loadAll,
    } as never);

    vi.mocked(sendChatCommand).mockResolvedValue({
      action: "resource_updated",
      message: "Resource updated.",
    });
  });

  it("ignores blank messages", async () => {
    await useChatStore.getState().sendMessage("   ");

    const state = useChatStore.getState();
    expect(sendChatCommand).not.toHaveBeenCalled();
    expect(state.messages).toHaveLength(1);
    expect(state.isSending).toBe(false);
    expect(state.error).toBeNull();
  });

  it("sets sending state while awaiting a response", async () => {
    let resolveCommand!: (value: Awaited<ReturnType<typeof sendChatCommand>>) => void;
    const commandPromise = new Promise<Awaited<ReturnType<typeof sendChatCommand>>>((resolve) => {
      resolveCommand = resolve;
    });

    vi.mocked(sendChatCommand).mockReturnValueOnce(commandPromise);

    const sendPromise = useChatStore.getState().sendMessage("list resources");

    const midState = useChatStore.getState();
    expect(midState.isSending).toBe(true);
    expect(midState.error).toBeNull();
    expect(midState.messages).toHaveLength(2);
    expect(midState.messages[1]?.role).toBe("user");

    resolveCommand({
      action: "list_resources",
      message: "Resources listed.",
    });
    await sendPromise;

    const finalState = useChatStore.getState();
    expect(finalState.isSending).toBe(false);
    expect(finalState.messages).toHaveLength(3);
  });

  it("sends mutation commands and refreshes resource/task stores", async () => {
    await useChatStore.getState().sendMessage("update resource https://example.com");

    const state = useChatStore.getState();
    expect(sendChatCommand).toHaveBeenCalledWith("update resource https://example.com");
    expect(state.messages).toHaveLength(3);
    expect(state.messages[1]?.role).toBe("user");
    expect(state.messages[2]?.role).toBe("assistant");
    expect(state.messages[2]?.content).toBe("Resource updated.");
    expect(state.isSending).toBe(false);
    expect(state.error).toBeNull();
    expect(loadResources).toHaveBeenCalledWith({ silent: true });
    expect(loadAll).toHaveBeenCalledWith({ silent: true });
  });

  it("trims outgoing chat content", async () => {
    vi.mocked(sendChatCommand).mockResolvedValueOnce({
      action: "list_resources",
      message: "Resources listed.",
    });

    await useChatStore.getState().sendMessage("  list resources  ");

    const state = useChatStore.getState();
    expect(sendChatCommand).toHaveBeenCalledWith("list resources");
    expect(state.messages[1]?.content).toBe("list resources");
    expect(loadResources).not.toHaveBeenCalled();
    expect(loadAll).not.toHaveBeenCalled();
  });

  it("does not refresh stores for non-mutation actions", async () => {
    vi.mocked(sendChatCommand).mockResolvedValueOnce({
      action: "list_resources",
      message: "  Resources listed.  ",
    });

    await useChatStore.getState().sendMessage("list resources");

    const state = useChatStore.getState();
    expect(state.messages[2]?.content).toBe("Resources listed.");
    expect(loadResources).not.toHaveBeenCalled();
    expect(loadAll).not.toHaveBeenCalled();
  });

  it("uses a fallback response when the command result has no message", async () => {
    vi.mocked(sendChatCommand).mockResolvedValueOnce({
      action: "todo_created",
      message: "  ",
    });

    await useChatStore.getState().sendMessage("create todo Test");

    const state = useChatStore.getState();
    expect(state.messages[2]?.content).toBe("Command executed (todo_created).");
    expect(loadResources).toHaveBeenCalledWith({ silent: true });
    expect(loadAll).toHaveBeenCalledWith({ silent: true });
  });

  it("surfaces chat command failures", async () => {
    vi.mocked(sendChatCommand).mockRejectedValueOnce(new Error("mock chat error"));

    await useChatStore.getState().sendMessage("list resources");

    const state = useChatStore.getState();
    expect(state.messages).toHaveLength(3);
    expect(state.messages[2]?.content).toBe("Command failed: mock chat error");
    expect(state.isSending).toBe(false);
    expect(state.error).toBe("mock chat error");
    expect(loadResources).not.toHaveBeenCalled();
    expect(loadAll).not.toHaveBeenCalled();
  });

  it("uses a fallback error message for non-error rejections", async () => {
    vi.mocked(sendChatCommand).mockRejectedValueOnce("mock chat error");

    await useChatStore.getState().sendMessage("list resources");

    const state = useChatStore.getState();
    expect(state.messages[2]?.content).toBe("Command failed: Failed to run chat command");
    expect(state.isSending).toBe(false);
    expect(state.error).toBe("Failed to run chat command");
    expect(loadResources).not.toHaveBeenCalled();
    expect(loadAll).not.toHaveBeenCalled();
  });
});
