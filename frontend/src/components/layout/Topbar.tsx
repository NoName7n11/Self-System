import { useResourceStore } from "../../stores/useResourceStore";
import { useSyncStore } from "../../stores/useSyncStore";
import type { SyncStatus } from "../../types";

interface TopbarProps {
  totalResources: number;
  visibleResources: number;
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

  const statusMap: Record<SyncStatus, string> = {
    idle: "Idle",
    connecting: "Connecting",
    connected: "Connected",
    reconnecting: reconnectAttempt > 0 ? `Reconnecting (${reconnectAttempt})` : "Reconnecting",
    offline: "Offline",
  };

  const syncLabel = statusMap[syncStatus];
  const pollingLabel = fallbackPolling ? " • Polling" : "";
  const statusLabel = isLoading ? `${syncLabel}${pollingLabel} • Refreshing` : `${syncLabel}${pollingLabel}`;
  const pollingClass = fallbackPolling ? "is-polling" : "";
  const runtimeClass =
    syncStatus === "connected"
      ? "is-connected"
      : syncStatus === "reconnecting" || syncStatus === "connecting"
        ? "is-reconnecting"
        : syncStatus === "offline"
          ? "is-offline"
          : "is-idle";

  const runtimeBaseTitle =
    syncError?.trim() || (lastEventType?.trim() ? `Last sync event: ${lastEventType.trim()}` : "Sync status indicator");
  const runtimeTitle = fallbackPolling ? `${runtimeBaseTitle} (fallback polling active)` : runtimeBaseTitle;

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
