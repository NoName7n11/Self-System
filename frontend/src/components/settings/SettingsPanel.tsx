import { useMemo } from "react";

import { API_BASE_URL, DEFAULT_SYNC_WS_PATH, resolveSyncWebSocketURL } from "../../api/client";
import { useResourceStore } from "../../stores/useResourceStore";
import { useSyncStore } from "../../stores/useSyncStore";
import { useTaskStore } from "../../stores/useTaskStore";

function firstNonEmpty(...values: string[]): string {
  for (const value of values) {
    const trimmed = value.trim();
    if (trimmed !== "") {
      return trimmed;
    }
  }
  return "N/A";
}

export default function SettingsPanel() {
  const resourceCount = useResourceStore((state) => state.resources.length);
  const todoCount = useTaskStore((state) => state.todos.length);
  const reminderCount = useTaskStore((state) => state.reminders.length);
  const syncStatus = useSyncStore((state) => state.status);
  const reconnectAttempt = useSyncStore((state) => state.reconnectAttempt);
  const fallbackPolling = useSyncStore((state) => state.fallbackPolling);
  const lastEventType = useSyncStore((state) => state.lastEventType);
  const syncError = useSyncStore((state) => state.error);

  const resolvedSyncURL = useMemo(() => {
    try {
      return resolveSyncWebSocketURL();
    } catch {
      return "N/A";
    }
  }, []);

  return (
    <section className="settings-panel panel">
      <div className="panel-heading">
        <h2>Runtime Settings</h2>
        <p>Read-only operational context for API, sync transport, and loaded workspace entities.</p>
      </div>

      <div className="settings-grid">
        <article className="settings-card">
          <h3>Endpoints</h3>
          <div className="settings-row">
            <span>API base URL</span>
            <strong>{firstNonEmpty(API_BASE_URL)}</strong>
          </div>
          <div className="settings-row">
            <span>Sync websocket</span>
            <strong>{firstNonEmpty(resolvedSyncURL)}</strong>
          </div>
          <div className="settings-row">
            <span>Default sync path</span>
            <strong>{firstNonEmpty(DEFAULT_SYNC_WS_PATH)}</strong>
          </div>
        </article>

        <article className="settings-card">
          <h3>Sync Runtime</h3>
          <div className="settings-row">
            <span>Status</span>
            <strong>{syncStatus}</strong>
          </div>
          <div className="settings-row">
            <span>Reconnect attempts</span>
            <strong>{String(reconnectAttempt)}</strong>
          </div>
          <div className="settings-row">
            <span>Fallback polling</span>
            <strong>{fallbackPolling ? "active" : "inactive"}</strong>
          </div>
          <div className="settings-row">
            <span>Last event</span>
            <strong>{firstNonEmpty(lastEventType || "")}</strong>
          </div>
          <div className="settings-row">
            <span>Error</span>
            <strong>{firstNonEmpty(syncError || "")}</strong>
          </div>
        </article>

        <article className="settings-card">
          <h3>Loaded Records</h3>
          <div className="settings-row">
            <span>Resources</span>
            <strong>{String(resourceCount)}</strong>
          </div>
          <div className="settings-row">
            <span>Todos</span>
            <strong>{String(todoCount)}</strong>
          </div>
          <div className="settings-row">
            <span>Reminders</span>
            <strong>{String(reminderCount)}</strong>
          </div>
        </article>
      </div>
    </section>
  );
}
