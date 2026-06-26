import { IcoBell } from "../icons";
import { DEMO_NOTIFS } from "../../lib/demoData";
import { useLayoutStore } from "../../stores/useLayoutStore";
import type { GraphView } from "../../types";

interface TopbarProps {
  resourceCount: number;
  connectionCount?: number;
}

const VIEWS: Array<{ key: GraphView; label: string }> = [
  { key: "graph", label: "GRAPH" },
  { key: "map", label: "MAP" },
  { key: "progress", label: "PROGRESS" },
];

// kept exported for any remaining importers
export function getSyncStatusLabel(): string { return ""; }
export function getRuntimeClass(): string { return ""; }
export function getRuntimeTitle(): string { return ""; }

export default function Topbar({ resourceCount, connectionCount = 342 }: TopbarProps) {
  const view = useLayoutStore((s) => s.view);
  const setView = useLayoutStore((s) => s.setView);
  const notifOpen = useLayoutStore((s) => s.notifOpen);
  const notifSeen = useLayoutStore((s) => s.notifSeen);
  const notifMuted = useLayoutStore((s) => s.notifMuted);
  const toggleNotif = useLayoutStore((s) => s.toggleNotif);
  const closeNotif = useLayoutStore((s) => s.closeNotif);
  const toggleMute = useLayoutStore((s) => s.toggleMute);

  return (
    <header className="top-bar">
      <div>
        <div className="top-bar-title-main">KNOWLEDGE GRAPH</div>
        <div className="top-bar-title-sub">{resourceCount} RESOURCES · {connectionCount} CONNECTIONS</div>
      </div>

      <div className="top-bar-spacer" />

      {/* notifications */}
      <div className="notif-wrap">
        <button className="notif-bell" onClick={toggleNotif} type="button" aria-label="Notifications">
          <IcoBell />
          {!notifSeen && <span className="notif-dot" />}
        </button>
        {notifOpen && (
          <>
            <div className="notif-scrim" onClick={closeNotif} />
            <div className="notif-panel">
              <div className="notif-head">
                <IcoBell />
                <span className="notif-head-label">NOTIFICATIONS</span>
                <span className="notif-head-count">{DEMO_NOTIFS.length}</span>
                <button className="notif-head-btn" onClick={toggleMute} type="button">{notifMuted ? "UNMUTE" : "MUTE"}</button>
                <button className="notif-head-btn is-danger" onClick={closeNotif} type="button">CLEAR</button>
              </div>
              <div className="notif-list">
                {DEMO_NOTIFS.map((n) => (
                  <div key={n.id} className="notif-row">
                    <span className="notif-ico" style={{ background: n.color }} />
                    <div className="notif-body">
                      <div className="notif-row-top">
                        <span className="notif-title">{n.title}</span>
                        <span className="notif-time">{n.time}</span>
                      </div>
                      <div className="notif-text">{n.body}</div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </>
        )}
      </div>

      {/* view switch */}
      <div className="view-switch" role="group" aria-label="View mode">
        {VIEWS.map(({ key, label }) => (
          <button key={key} className={`view-switch-btn${view === key ? " is-active" : ""}`}
            onClick={() => setView(key)} type="button">
            {label}
          </button>
        ))}
      </div>
    </header>
  );
}
