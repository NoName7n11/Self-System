import { beforeEach, afterEach, describe, expect, it, vi } from "vitest";

import { resolveSyncWebSocketURL } from "../api/client";
import { useResourceStore } from "./useResourceStore";
import { useTaskStore } from "./useTaskStore";
import { useSyncStore } from "./useSyncStore";

vi.mock("../api/client", () => ({
  resolveSyncWebSocketURL: vi.fn(),
}));

class FakeWebSocket {
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSING = 2;
  static CLOSED = 3;

  static instances: FakeWebSocket[] = [];

  readyState = FakeWebSocket.CONNECTING;
  onopen: ((event: Event) => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;
  onclose: ((event: CloseEvent) => void) | null = null;

  constructor(public readonly url: string) {
    FakeWebSocket.instances.push(this);
  }

  open() {
    this.readyState = FakeWebSocket.OPEN;
    this.onopen?.(new Event("open"));
  }

  emitMessage(data: string) {
    this.onmessage?.({ data } as MessageEvent);
  }

  emitError() {
    this.onerror?.(new Event("error"));
  }

  close(code = 1000, reason = "") {
    this.readyState = FakeWebSocket.CLOSED;
    this.onclose?.({ code, reason, wasClean: code === 1000 } as CloseEvent);
  }
}

function resetSyncStore() {
  useSyncStore.setState({
    status: "idle",
    error: null,
    lastEventType: null,
    lastSequence: null,
    reconnectAttempt: 0,
    fallbackPolling: false,
  });
}

describe("useSyncStore", () => {
  const loadResources = vi.fn();
  const loadAll = vi.fn();

  beforeEach(() => {
    vi.useFakeTimers();
    vi.stubGlobal("window", globalThis);
    vi.stubGlobal("WebSocket", FakeWebSocket as never);

    FakeWebSocket.instances = [];
    resetSyncStore();

    loadResources.mockReset();
    loadAll.mockReset();
    loadResources.mockResolvedValue(undefined);
    loadAll.mockResolvedValue(undefined);

    vi.spyOn(useResourceStore, "getState").mockReturnValue({
      loadResources,
    } as never);
    vi.spyOn(useTaskStore, "getState").mockReturnValue({
      loadAll,
    } as never);

    vi.mocked(resolveSyncWebSocketURL).mockReturnValue("ws://example.test/sync");
  });

  afterEach(() => {
    useSyncStore.getState().stop();
    vi.clearAllTimers();
    vi.useRealTimers();
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("enters fallback polling and reloads immediately when websocket URL resolution fails", () => {
    vi.mocked(resolveSyncWebSocketURL).mockImplementation(() => {
      throw new Error("missing sync url");
    });

    useSyncStore.getState().start();

    const state = useSyncStore.getState();
    expect(state.status).toBe("offline");
    expect(state.error).toBe("missing sync url");
    expect(state.fallbackPolling).toBe(true);
    expect(FakeWebSocket.instances).toHaveLength(0);
    expect(loadResources).toHaveBeenCalledWith({ silent: true });
    expect(loadAll).toHaveBeenCalledWith({ silent: true });
  });

  it("treats a blank websocket URL as offline and starts fallback polling", () => {
    vi.mocked(resolveSyncWebSocketURL).mockReturnValue("   ");

    useSyncStore.getState().start();

    const state = useSyncStore.getState();
    expect(state.status).toBe("offline");
    expect(state.error).toBe("Sync websocket URL is empty");
    expect(state.fallbackPolling).toBe(true);
    expect(FakeWebSocket.instances).toHaveLength(0);
    expect(loadResources).toHaveBeenCalledWith({ silent: true });
    expect(loadAll).toHaveBeenCalledWith({ silent: true });
  });

  it("reconnects after a websocket close and refreshes data during fallback polling", () => {
    useSyncStore.getState().start();

    const firstSocket = FakeWebSocket.instances[0];
    expect(firstSocket.url).toBe("ws://example.test/sync");

    firstSocket.open();
    expect(useSyncStore.getState().status).toBe("connected");
    expect(useSyncStore.getState().fallbackPolling).toBe(false);

    loadResources.mockClear();
    loadAll.mockClear();

    firstSocket.close(1006, "network lost");

    const afterClose = useSyncStore.getState();
    expect(afterClose.status).toBe("reconnecting");
    expect(afterClose.reconnectAttempt).toBe(1);
    expect(afterClose.error).toBe("Sync disconnected (1006): network lost");
    expect(afterClose.fallbackPolling).toBe(true);
    expect(loadResources).toHaveBeenCalledWith({ silent: true });
    expect(loadAll).toHaveBeenCalledWith({ silent: true });

    expect(FakeWebSocket.instances).toHaveLength(1);
    vi.advanceTimersByTime(1199);
    expect(FakeWebSocket.instances).toHaveLength(1);
    vi.advanceTimersByTime(1);

    expect(FakeWebSocket.instances).toHaveLength(2);
    const secondSocket = FakeWebSocket.instances[1];
    expect(secondSocket?.url).toBe("ws://example.test/sync");

    expect(useSyncStore.getState()).toMatchObject({
      status: "reconnecting",
      reconnectAttempt: 1,
      fallbackPolling: true,
      error: null,
    });

    secondSocket?.open();

    const recovered = useSyncStore.getState();
    expect(recovered.status).toBe("connected");
    expect(recovered.error).toBeNull();
    expect(recovered.reconnectAttempt).toBe(0);
    expect(recovered.fallbackPolling).toBe(false);
  });

  it("reports unknown close codes while reconnecting and recovers after reconnect open", () => {
    useSyncStore.getState().start();

    const firstSocket = FakeWebSocket.instances[0];
    firstSocket.open();

    firstSocket.close(Number.NaN, "network lost");

    expect(useSyncStore.getState()).toMatchObject({
      status: "reconnecting",
      reconnectAttempt: 1,
      fallbackPolling: true,
      error: "Sync disconnected (unknown): network lost",
    });

    vi.advanceTimersByTime(1200);

    expect(FakeWebSocket.instances).toHaveLength(2);
    const secondSocket = FakeWebSocket.instances[1];
    secondSocket?.open();

    expect(useSyncStore.getState()).toMatchObject({
      status: "connected",
      error: null,
      reconnectAttempt: 0,
      fallbackPolling: false,
    });
  });

  it("clears disconnect errors after a successful reconnect", () => {
    useSyncStore.getState().start();

    const firstSocket = FakeWebSocket.instances[0];
    firstSocket.open();

    firstSocket.close(1006, "network lost");

    const afterClose = useSyncStore.getState();
    expect(afterClose.status).toBe("reconnecting");
    expect(afterClose.error).toBe("Sync disconnected (1006): network lost");

    vi.advanceTimersByTime(1200);

    const secondSocket = FakeWebSocket.instances[1];
    secondSocket?.open();

    const recovered = useSyncStore.getState();
    expect(recovered.status).toBe("connected");
    expect(recovered.error).toBeNull();
    expect(recovered.reconnectAttempt).toBe(0);
    expect(recovered.fallbackPolling).toBe(false);
  });

  it("falls back when the websocket constructor throws", () => {
    class ThrowingWebSocket {
      constructor() {
        throw new Error("socket open failure");
      }
    }

    vi.stubGlobal("WebSocket", ThrowingWebSocket as never);

    useSyncStore.getState().start();

    expect(FakeWebSocket.instances).toHaveLength(0);
    expect(useSyncStore.getState()).toMatchObject({
      status: "offline",
      error: "socket open failure",
      fallbackPolling: true,
    });
    expect(loadResources).toHaveBeenCalledTimes(2);
    expect(loadAll).toHaveBeenCalledTimes(2);
  });

  it("omits the close reason when a websocket closes without one", () => {
    useSyncStore.getState().start();

    const firstSocket = FakeWebSocket.instances[0];
    firstSocket.open();

    firstSocket.close(1006);

    expect(useSyncStore.getState()).toMatchObject({
      status: "reconnecting",
      reconnectAttempt: 1,
      fallbackPolling: true,
      error: "Sync disconnected (1006)",
    });

    vi.advanceTimersByTime(1200);

    expect(FakeWebSocket.instances).toHaveLength(2);
    FakeWebSocket.instances[1]?.open();

    expect(useSyncStore.getState()).toMatchObject({
      status: "connected",
      error: null,
      reconnectAttempt: 0,
      fallbackPolling: false,
    });
  });

  it("applies increasing reconnect backoff and caps it at the maximum delay", () => {
    useSyncStore.getState().start();

    const expectedDelays = [1200, 2400, 4800, 9600, 12000];
    let socket = FakeWebSocket.instances[0];

    expectedDelays.forEach((delay, index) => {
      socket.close(1006, `reconnect cycle ${index + 1}`);

      expect(useSyncStore.getState()).toMatchObject({
        status: "reconnecting",
        reconnectAttempt: index + 1,
        fallbackPolling: true,
        error: `Sync disconnected (1006): reconnect cycle ${index + 1}`,
      });

      const beforeCount = FakeWebSocket.instances.length;
      vi.advanceTimersByTime(delay - 1);
      expect(FakeWebSocket.instances).toHaveLength(beforeCount);

      vi.advanceTimersByTime(1);
      expect(FakeWebSocket.instances).toHaveLength(beforeCount + 1);

      socket = FakeWebSocket.instances[beforeCount];
    });

    expect(useSyncStore.getState()).toMatchObject({
      status: "reconnecting",
      reconnectAttempt: expectedDelays.length,
      fallbackPolling: true,
      error: null,
    });
    expect(FakeWebSocket.instances).toHaveLength(expectedDelays.length + 1);
  });

  it("reuses the existing websocket while connecting or connected", () => {
    useSyncStore.getState().start();

    const firstSocket = FakeWebSocket.instances[0];
    expect(firstSocket.url).toBe("ws://example.test/sync");

    loadResources.mockClear();
    loadAll.mockClear();

    useSyncStore.getState().start();

    expect(FakeWebSocket.instances).toHaveLength(1);
    expect(useSyncStore.getState().status).toBe("connecting");
    expect(loadResources).toHaveBeenCalledTimes(1);
    expect(loadAll).toHaveBeenCalledTimes(1);

    firstSocket.open();

    loadResources.mockClear();
    loadAll.mockClear();

    useSyncStore.getState().start();

    expect(FakeWebSocket.instances).toHaveLength(1);
    expect(useSyncStore.getState().status).toBe("connected");
    expect(loadResources).toHaveBeenCalledTimes(1);
    expect(loadAll).toHaveBeenCalledTimes(1);
  });

  it("stops fallback polling after a successful connection", () => {
    useSyncStore.getState().start();

    const socket = FakeWebSocket.instances[0];
    expect(useSyncStore.getState().fallbackPolling).toBe(true);

    loadResources.mockClear();
    loadAll.mockClear();

    socket.open();

    expect(useSyncStore.getState()).toMatchObject({
      status: "connected",
      fallbackPolling: false,
    });

    vi.advanceTimersByTime(12000);

    expect(loadResources).not.toHaveBeenCalled();
    expect(loadAll).not.toHaveBeenCalled();
  });

  it("restarts with a fresh websocket after stop", () => {
    useSyncStore.getState().start();

    const firstSocket = FakeWebSocket.instances[0];
    firstSocket.open();

    useSyncStore.getState().stop();

    expect(useSyncStore.getState()).toMatchObject({
      status: "idle",
      error: null,
      reconnectAttempt: 0,
      fallbackPolling: false,
    });
    expect(firstSocket.readyState).toBe(FakeWebSocket.CLOSED);

    useSyncStore.getState().start();

    expect(FakeWebSocket.instances).toHaveLength(2);
    const restartedSocket = FakeWebSocket.instances[1];
    expect(restartedSocket?.url).toBe("ws://example.test/sync");
    expect(useSyncStore.getState().status).toBe("connecting");

    restartedSocket?.open();

    expect(useSyncStore.getState()).toMatchObject({
      status: "connected",
      error: null,
      reconnectAttempt: 0,
      fallbackPolling: false,
    });
  });

  it("debounces sync mutation events into silent resource and task refreshes", () => {
    useSyncStore.getState().start();

    const socket = FakeWebSocket.instances[0];
    socket.open();

    loadResources.mockClear();
    loadAll.mockClear();

    socket.emitMessage(JSON.stringify({ type: "sync.resource.updated", sequence: 41 }));
    socket.emitMessage(JSON.stringify({ type: "sync.todo.updated", sequence: 42 }));

    const state = useSyncStore.getState();
    expect(state.lastEventType).toBe("sync.todo.updated");
    expect(state.lastSequence).toBe(42);

    vi.advanceTimersByTime(319);
    expect(loadResources).not.toHaveBeenCalled();
    expect(loadAll).not.toHaveBeenCalled();

    vi.advanceTimersByTime(1);
    expect(loadResources).toHaveBeenCalledWith({ silent: true });
    expect(loadAll).toHaveBeenCalledWith({ silent: true });
  });

  it("refreshes only resources for resource mutation events", () => {
    useSyncStore.getState().start();

    const socket = FakeWebSocket.instances[0];
    socket.open();

    loadResources.mockClear();
    loadAll.mockClear();

    socket.emitMessage(JSON.stringify({ type: "sync.resource.updated", sequence: 51 }));

    vi.advanceTimersByTime(320);

    expect(loadResources).toHaveBeenCalledWith({ silent: true });
    expect(loadAll).not.toHaveBeenCalled();
  });

  it("refreshes only tasks for task mutation events", () => {
    useSyncStore.getState().start();

    const socket = FakeWebSocket.instances[0];
    socket.open();

    loadResources.mockClear();
    loadAll.mockClear();

    socket.emitMessage(JSON.stringify({ type: "sync.todo.updated", sequence: 52 }));

    vi.advanceTimersByTime(320);

    expect(loadAll).toHaveBeenCalledWith({ silent: true });
    expect(loadResources).not.toHaveBeenCalled();
  });

  it("preserves lastSequence when a sync event has an invalid sequence", () => {
    useSyncStore.getState().start();

    const socket = FakeWebSocket.instances[0];
    socket.open();

    loadResources.mockClear();
    loadAll.mockClear();

    socket.emitMessage(JSON.stringify({ type: "sync.heartbeat", sequence: 11 }));
    socket.emitMessage(JSON.stringify({ type: "sync.notice", sequence: "bad" }));

    const state = useSyncStore.getState();
    expect(state.lastEventType).toBe("sync.notice");
    expect(state.lastSequence).toBe(11);

    vi.advanceTimersByTime(320);
    expect(loadResources).not.toHaveBeenCalled();
    expect(loadAll).not.toHaveBeenCalled();
  });

  it("ignores events with missing or blank type", () => {
    useSyncStore.getState().start();

    const socket = FakeWebSocket.instances[0];
    socket.open();

    socket.emitMessage(JSON.stringify({ type: "sync.heartbeat", sequence: 7 }));

    loadResources.mockClear();
    loadAll.mockClear();

    socket.emitMessage(JSON.stringify({ type: "   ", sequence: 9 }));
    socket.emitMessage(JSON.stringify({ sequence: 10 }));
    socket.emitMessage("");

    const state = useSyncStore.getState();
    expect(state.lastEventType).toBe("sync.heartbeat");
    expect(state.lastSequence).toBe(7);

    vi.advanceTimersByTime(320);
    expect(loadResources).not.toHaveBeenCalled();
    expect(loadAll).not.toHaveBeenCalled();
  });

  it("stops cleanly and cancels pending reconnect and refresh work", () => {
    useSyncStore.getState().start();

    const socket = FakeWebSocket.instances[0];
    socket.open();
    socket.emitMessage(JSON.stringify({ type: "sync.resource.updated", sequence: 41 }));
    socket.emitMessage(JSON.stringify({ type: "sync.todo.updated", sequence: 42 }));
    socket.close(1006, "network lost");

    loadResources.mockClear();
    loadAll.mockClear();

    useSyncStore.getState().stop();
    socket.emitError();

    const state = useSyncStore.getState();
    expect(state.status).toBe("idle");
    expect(state.error).toBeNull();
    expect(state.reconnectAttempt).toBe(0);
    expect(state.fallbackPolling).toBe(false);
    expect(socket.readyState).toBe(FakeWebSocket.CLOSED);

    vi.advanceTimersByTime(319);
    expect(loadResources).not.toHaveBeenCalled();
    expect(loadAll).not.toHaveBeenCalled();

    vi.advanceTimersByTime(12000);
    expect(FakeWebSocket.instances).toHaveLength(1);
    expect(loadResources).not.toHaveBeenCalled();
    expect(loadAll).not.toHaveBeenCalled();
  });

  it("ignores malformed and non-mutation websocket messages", () => {
    useSyncStore.getState().start();

    const socket = FakeWebSocket.instances[0];
    socket.open();

    loadResources.mockClear();
    loadAll.mockClear();

    socket.emitMessage("not json");
    socket.emitMessage(JSON.stringify({ type: "sync.heartbeat", sequence: 7 }));

    const state = useSyncStore.getState();
    expect(state.lastEventType).toBe("sync.heartbeat");
    expect(state.lastSequence).toBe(7);
    expect(loadResources).not.toHaveBeenCalled();
    expect(loadAll).not.toHaveBeenCalled();
  });

  it("surfaces websocket transport errors without disconnecting immediately", () => {
    useSyncStore.getState().start();

    const socket = FakeWebSocket.instances[0];
    socket.open();

    socket.emitError();

    const state = useSyncStore.getState();
    expect(state.status).toBe("connected");
    expect(state.error).toBe("Sync transport error detected");
    expect(state.fallbackPolling).toBe(false);
  });

  it("reuses fallback polling and keeps reloading while offline when start is called again", () => {
    vi.mocked(resolveSyncWebSocketURL).mockReturnValue("   ");

    useSyncStore.getState().start();

    loadResources.mockClear();
    loadAll.mockClear();

    useSyncStore.getState().start();

    expect(FakeWebSocket.instances).toHaveLength(0);
    expect(useSyncStore.getState()).toMatchObject({
      status: "offline",
      fallbackPolling: true,
      error: "Sync websocket URL is empty",
    });
    expect(loadResources).toHaveBeenCalledTimes(2);
    expect(loadAll).toHaveBeenCalledTimes(2);

    vi.advanceTimersByTime(12000);

    expect(loadResources).toHaveBeenCalledTimes(3);
    expect(loadAll).toHaveBeenCalledTimes(3);
  });

  it("keeps polling for refreshes while disconnected", () => {
    useSyncStore.getState().start();

    const socket = FakeWebSocket.instances[0];
    socket.open();

    loadResources.mockClear();
    loadAll.mockClear();

    socket.close(1006, "network lost");

    expect(useSyncStore.getState().fallbackPolling).toBe(true);

    loadResources.mockClear();
    loadAll.mockClear();

    vi.advanceTimersByTime(12000);

    expect(loadResources).toHaveBeenCalledTimes(1);
    expect(loadAll).toHaveBeenCalledTimes(1);
  });
});