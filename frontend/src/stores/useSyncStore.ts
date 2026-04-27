import { create } from "zustand";

import { resolveSyncWebSocketURL } from "../api/client";
import type { SyncEvent, SyncStatus } from "../types";
import { useResourceStore } from "./useResourceStore";
import { useTaskStore } from "./useTaskStore";

const reconnectBaseDelayMs = 1200;
const reconnectMaxDelayMs = 12000;
const mutationRefreshDebounceMs = 320;
const fallbackPollIntervalMs = 12000;

const resourceMutationEventTypes = new Set<string>([
  "sync.resource.created",
  "sync.resource.updated",
  "sync.resource.deleted",
  "sync.category.updated",
]);

const taskMutationEventTypes = new Set<string>([
  "sync.todo.updated",
  "sync.reminder.updated",
]);

let socket: WebSocket | null = null;
let reconnectTimer: number | null = null;
let refreshTimer: number | null = null;
let taskRefreshTimer: number | null = null;
let fallbackPollTimer: number | null = null;
let stopRequested = false;

function clearReconnectTimer() {
  if (reconnectTimer !== null) {
    window.clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
}

function clearRefreshTimer() {
  if (refreshTimer !== null) {
    window.clearTimeout(refreshTimer);
    refreshTimer = null;
  }
}

function clearTaskRefreshTimer() {
  if (taskRefreshTimer !== null) {
    window.clearTimeout(taskRefreshTimer);
    taskRefreshTimer = null;
  }
}

function scheduleBackgroundReload() {
  void useResourceStore.getState().loadResources({ silent: true });
  void useTaskStore.getState().loadAll({ silent: true });
}

function stopFallbackPolling(set: (partial: Partial<SyncState>) => void) {
  if (fallbackPollTimer !== null) {
    window.clearInterval(fallbackPollTimer);
    fallbackPollTimer = null;
  }
  set({ fallbackPolling: false });
}

function startFallbackPolling(
  get: () => SyncState,
  set: (partial: Partial<SyncState>) => void,
  immediate = false,
) {
  if (stopRequested) {
    return;
  }

  if (fallbackPollTimer !== null) {
    set({ fallbackPolling: true });
    if (immediate) {
      scheduleBackgroundReload();
    }
    return;
  }

  set({ fallbackPolling: true });
  if (immediate) {
    scheduleBackgroundReload();
  }

  fallbackPollTimer = window.setInterval(() => {
    const state = get();
    if (state.status === "connected" || stopRequested) {
      stopFallbackPolling(set);
      return;
    }
    scheduleBackgroundReload();
  }, fallbackPollIntervalMs);
}

function normalizeCode(event: CloseEvent): string {
  if (typeof event.code === "number" && Number.isFinite(event.code)) {
    return String(event.code);
  }
  return "unknown";
}

function parseSyncEvent(data: unknown): SyncEvent | null {
  if (typeof data !== "string" || data.trim() === "") {
    return null;
  }

  try {
    const parsed = JSON.parse(data) as Record<string, unknown>;
    const type = typeof parsed.type === "string" ? parsed.type.trim() : "";
    if (type === "") {
      return null;
    }

    const sequence = typeof parsed.sequence === "number" && Number.isFinite(parsed.sequence) ? parsed.sequence : undefined;
    const payload =
      parsed.payload && typeof parsed.payload === "object" && !Array.isArray(parsed.payload)
        ? (parsed.payload as Record<string, unknown>)
        : undefined;

    return {
      type,
      sequence,
      payload,
      timestamp: typeof parsed.timestamp === "string" ? parsed.timestamp : undefined,
    };
  } catch {
    return null;
  }
}

function scheduleResourceRefresh() {
  if (refreshTimer !== null) {
    return;
  }

  refreshTimer = window.setTimeout(() => {
    refreshTimer = null;
    void useResourceStore.getState().loadResources({ silent: true });
  }, mutationRefreshDebounceMs);
}

function scheduleTaskRefresh() {
  if (taskRefreshTimer !== null) {
    return;
  }

  taskRefreshTimer = window.setTimeout(() => {
    taskRefreshTimer = null;
    void useTaskStore.getState().loadAll({ silent: true });
  }, mutationRefreshDebounceMs);
}

interface SyncState {
  status: SyncStatus;
  error: string | null;
  lastEventType: string | null;
  lastSequence: number | null;
  reconnectAttempt: number;
  fallbackPolling: boolean;
  start: () => void;
  stop: () => void;
}

function connect(get: () => SyncState, set: (partial: Partial<SyncState>) => void) {
  if (stopRequested) {
    return;
  }

  if (socket && (socket.readyState === WebSocket.OPEN || socket.readyState === WebSocket.CONNECTING)) {
    return;
  }

  const nextStatus: SyncStatus = get().reconnectAttempt > 0 ? "reconnecting" : "connecting";
  set({ status: nextStatus, error: null });
  startFallbackPolling(get, set, false);

  let syncURL = "";
  try {
    syncURL = resolveSyncWebSocketURL();
  } catch (error) {
    const message = error instanceof Error ? error.message : "Failed to resolve sync websocket URL";
    set({ status: "offline", error: message });
    startFallbackPolling(get, set, true);
    return;
  }

  if (syncURL.trim() === "") {
    set({ status: "offline", error: "Sync websocket URL is empty" });
    startFallbackPolling(get, set, true);
    return;
  }

  try {
    socket = new WebSocket(syncURL);
  } catch (error) {
    const message = error instanceof Error ? error.message : "Failed to open sync websocket";
    set({ status: "offline", error: message });
    startFallbackPolling(get, set, true);
    return;
  }

  socket.onopen = () => {
    clearReconnectTimer();
    stopFallbackPolling(set);
    set({ status: "connected", error: null, reconnectAttempt: 0 });
  };

  socket.onmessage = (event) => {
    const parsed = parseSyncEvent(event.data);
    if (!parsed) {
      return;
    }

    set({
      lastEventType: parsed.type,
      lastSequence: typeof parsed.sequence === "number" ? parsed.sequence : get().lastSequence,
    });

    if (resourceMutationEventTypes.has(parsed.type)) {
      scheduleResourceRefresh();
    }

    if (taskMutationEventTypes.has(parsed.type)) {
      scheduleTaskRefresh();
    }
  };

  socket.onerror = () => {
    if (stopRequested) {
      return;
    }
    set({ error: "Sync transport error detected" });
  };

  socket.onclose = (event) => {
    socket = null;

    if (stopRequested) {
      set({ status: "idle", error: null, reconnectAttempt: 0 });
      return;
    }

    const attempt = get().reconnectAttempt + 1;
    const delay = Math.min(reconnectBaseDelayMs * 2 ** (attempt - 1), reconnectMaxDelayMs);
    const reason = event.reason?.trim();
    const code = normalizeCode(event);

    set({
      status: "reconnecting",
      reconnectAttempt: attempt,
      error: reason ? `Sync disconnected (${code}): ${reason}` : `Sync disconnected (${code})`,
    });

    startFallbackPolling(get, set, true);

    clearReconnectTimer();
    reconnectTimer = window.setTimeout(() => {
      reconnectTimer = null;
      connect(get, set);
    }, delay);
  };
}

export const useSyncStore = create<SyncState>((set, get) => ({
  status: "idle",
  error: null,
  lastEventType: null,
  lastSequence: null,
  reconnectAttempt: 0,
  fallbackPolling: false,

  start: () => {
    stopRequested = false;
    startFallbackPolling(get, (partial) => set(partial), true);
    connect(get, (partial) => set(partial));
  },

  stop: () => {
    stopRequested = true;
    clearReconnectTimer();
    clearRefreshTimer();
    clearTaskRefreshTimer();
    stopFallbackPolling((partial) => set(partial));

    if (socket) {
      socket.close();
      socket = null;
    }

    set({
      status: "idle",
      error: null,
      reconnectAttempt: 0,
    });
  },
}));
