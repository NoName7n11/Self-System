import { useResourceStore } from "../../stores/useResourceStore";
import { useSyncStore } from "../../stores/useSyncStore";
import type { SyncStatus } from "../../types";

interface TopbarProps {
  totalResources: number;
  visibleResources: number;
}

export function getSyncStatusLabel(
  syncStatus: SyncStatus,
  reconnectAttempt: number,
  fallbackPolling: boolean,
  isLoading: boolean,
): string {
  const statusMap: Record<SyncStatus, string> = {
    idle: "Idle",
    connecting: "Connecting",
    connected: "Connected",
    reconnecting: reconnectAttempt > 0 ? `Reconnecting (${reconnectAttempt})` : "Reconnecting",
    offline: "Offline",
  };

  const syncLabel = statusMap[syncStatus];
  const pollingLabel = fallbackPolling ? " • Polling" : "";

  return isLoading ? `${syncLabel}${pollingLabel} • Refreshing` : `${syncLabel}${pollingLabel}`;
}

export function getRuntimeClass(syncStatus: SyncStatus): string {
  if (syncStatus === "connected") {
    return "is-connected";
  }
  if (syncStatus === "reconnecting" || syncStatus === "connecting") {
    return "is-reconnecting";
  }
  if (syncStatus === "offline") {
    return "is-offline";
  }
  return "is-idle";
}

export function getRuntimeTitle(
  syncError: string | null,
  lastEventType: string | null,
  fallbackPolling: boolean,
): string {
  const runtimeBaseTitle =
    syncError?.trim() || (lastEventType?.trim() ? `Last sync event: ${lastEventType.trim()}` : "Sync status indicator");
  return fallbackPolling ? `${runtimeBaseTitle} (fallback polling active)` : runtimeBaseTitle;
}

export default function Topbar({ totalResources, visibleResources }: TopbarProps) {
  const query = useResourceStore((state) => state.filters.query);
  const isLoading = useResourceStore((state) => state.isLoading);
  const setQuery = useResourceStore((state) => state.setQuery);
  const loadResources = useResourceStore((state) => state.loadResources);
  const syncStatus = useSyncStore((state) => state.status);
  const syncError = useSyncStore((state) => state.error);
  const reconnectAttempt = useSyncStore((state) => state.reconnectAttempt);
  const fallbackPolling = useSyncStore((state) => state.fallbackPolling);
  const lastEventType = useSyncStore((state) => state.lastEventType);

  const statusLabel = getSyncStatusLabel(syncStatus, reconnectAttempt, fallbackPolling, isLoading);
  const pollingClass = fallbackPolling ? "is-polling" : "";
  const runtimeClass = getRuntimeClass(syncStatus);
  const runtimeTitle = getRuntimeTitle(syncError, lastEventType, fallbackPolling);

  return (
    <header className="topbar panel">
      <div className="topbar-left">
        <h1>Phase 2 Interactive Console</h1>
        <p>
          Visible <strong>{visibleResources}</strong> of <strong>{totalResources}</strong> resources
        </p>
      </div>

      <div className="topbar-right">
        <label className="search-shell" htmlFor="resource-search">
          <span>Search</span>
          <input
            id="resource-search"
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Title, URL, summary, category"
            value={query}
          />
        </label>

        <button className="ghost-button" onClick={() => void loadResources()} type="button">
          Reload
        </button>

        <div className={`runtime-pill ${runtimeClass} ${pollingClass} ${isLoading ? "is-loading" : ""}`} title={runtimeTitle}>
          {statusLabel}
        </div>
      </div>
    </header>
  );
}
