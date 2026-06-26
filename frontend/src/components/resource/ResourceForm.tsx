import { useMemo, useState } from "react";

import { IcoChevL, IcoChevR, IcoSend } from "../icons";
import { useChatStore } from "../../stores/useChatStore";
import { useLayoutStore } from "../../stores/useLayoutStore";
import { useResourceStore } from "../../stores/useResourceStore";
import type { ResourceType } from "../../types";

// ── color helpers ─────────────────────────────────────────────────────────────

const CAT_COLORS: Record<string, string> = {
  research: "#5B9CF6",
  ai:       "#A98BF5",
  finance:  "#48C78E",
  people:   "#E5B567",
  sources:  "#56B6C2",
  archive:  "#E06C75",
};

const TYPE_COLORS: Record<ResourceType, string> = {
  pdf:   "#F67373",
  link:  "#48C78E",
  note:  "#5B9CF6",
  doc:   "#9B59F6",
  image: "#F6739B",
};

function catColor(name: string): string {
  const key = name.trim().toLowerCase();
  return CAT_COLORS[key] ?? "#5B9CF6";
}

function fmtDate(iso: string): string {
  const d = new Date(iso);
  return isNaN(d.getTime()) ? "" : d.toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" });
}

const SUGGESTED_QUESTIONS = [
  "What are the key takeaways?",
  "How does this relate to other resources?",
  "What questions does this leave open?",
];

// ── kept export for old test compatibility ────────────────────────────────────
export function getResourceFormCopy(_: unknown) {
  return { heading: "Inspector", subheading: "" };
}

// ── Inspector ─────────────────────────────────────────────────────────────────

export default function Inspector() {
  const rightOpen    = useLayoutStore((s) => s.rightOpen);
  const toggleRight  = useLayoutStore((s) => s.toggleRight);
  const openDockTab  = useLayoutStore((s) => s.openDockTab);

  const resources         = useResourceStore((s) => s.resources);
  const selectedId        = useResourceStore((s) => s.selectedResourceId);
  const deleteSelectedResource = useResourceStore((s) => s.deleteSelectedResource);

  const sendMessage = useChatStore((s) => s.sendMessage);

  const [askInput, setAskInput] = useState("");

  const resource = useMemo(
    () => resources.find((r) => r.id === selectedId) ?? null,
    [resources, selectedId],
  );

  const typeColor = resource ? (TYPE_COLORS[resource.type ?? "link"] ?? "#9A9AA0") : "#9A9AA0";
  const catCol    = resource ? catColor(resource.categoryName) : "#5B9CF6";

  const sendAsk = async (q: string) => {
    const v = q.trim();
    if (!v) return;
    setAskInput("");
    await sendMessage(resource ? `[${resource.title || resource.url}] ${v}` : v);
    openDockTab("chat");
  };

  return (
    <aside className={`right-rail${rightOpen ? " is-open" : ""}`}>
      {/* ── Collapsed strip ── */}
      {!rightOpen && (
        <div className="right-rail-collapsed">
          <button className="right-expand-btn" onClick={toggleRight} type="button" aria-label="Open inspector">
            <IcoChevL />
          </button>
          {resource && (
            <span className="right-cat-swatch" style={{ background: catCol }} />
          )}
        </div>
      )}

      {/* ── Expanded inspector ── */}
      {rightOpen && (
        <>
          <div className="inspector-header">
            <span className="inspector-title">Inspector</span>
            <button className="inspector-collapse-btn" onClick={toggleRight} type="button" aria-label="Close inspector">
              <IcoChevR />
            </button>
          </div>

          {!resource ? (
            <div style={{ padding: "20px 14px", color: "var(--text-faint)", fontSize: 11 }}>
              Select a node on the graph to inspect it.
            </div>
          ) : (
            <>
              <div className="inspector-body">
                {/* preview well */}
                <div className="inspector-preview">
                  <div className="inspector-preview-host">{resource.host || "—"}</div>
                  <div className="inspector-preview-type" style={{ color: typeColor }}>
                    {(resource.type ?? "link").toUpperCase()}
                  </div>
                  <div className="inspector-preview-cat" style={{ background: catCol }} />
                </div>

                {/* details */}
                <div className="inspector-section">
                  <div className="inspector-resource-title">
                    {resource.title || resource.url || "Untitled"}
                  </div>
                  <div className="inspector-meta-row">
                    <span className={`type-badge ${resource.type ?? "link"}`}>{(resource.type ?? "link").toUpperCase()}</span>
                    <span className="inspector-date">ADDED {fmtDate(resource.createdAt)}</span>
                  </div>
                  <div className="inspector-meta-row">
                    {resource.categoryName && (
                      <span className="tag-chip">{resource.categoryName}</span>
                    )}
                  </div>
                </div>

                {/* quick actions */}
                <div className="inspector-section">
                  <div className="inspector-actions">
                    <button
                      className="inspector-action-btn"
                      onClick={() => resource.url && window.open(resource.url, "_blank")}
                      type="button"
                    >
                      OPEN
                    </button>
                    <button className="inspector-action-btn" type="button">EDIT</button>
                    <button className="inspector-action-btn" type="button">LINK</button>
                    <button
                      className="inspector-action-btn is-danger"
                      onClick={() => void deleteSelectedResource()}
                      type="button"
                    >
                      DELETE
                    </button>
                  </div>
                </div>

                {/* connections — deferred */}
                <div className="inspector-section">
                  <div className="inspector-section-label">CONNECTIONS · 0</div>
                  <div className="inspector-empty">No connections yet.</div>
                </div>

                {/* AI summary */}
                {resource.summary && (
                  <div className="inspector-section">
                    <div className="inspector-section-label">AI SUMMARY</div>
                    <div className="inspector-summary-text">{resource.summary}</div>
                    {SUGGESTED_QUESTIONS.map((q) => (
                      <div
                        key={q}
                        className="suggested-q"
                        onClick={() => void sendAsk(q)}
                        role="button"
                        tabIndex={0}
                        onKeyDown={(e) => e.key === "Enter" && void sendAsk(q)}
                      >
                        <span className="suggested-q-arrow">›</span>
                        {q}
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {/* footer ask input */}
              <div className="inspector-footer">
                <input
                  className="inspector-ask-input"
                  value={askInput}
                  onChange={(e) => setAskInput(e.target.value)}
                  onKeyDown={(e) => { if (e.key === "Enter") void sendAsk(askInput); }}
                  placeholder="ASK ABOUT THIS RESOURCE…"
                />
                <button
                  className="inspector-send-btn"
                  onClick={() => void sendAsk(askInput)}
                  type="button"
                  aria-label="Send"
                >
                  <IcoSend />
                </button>
              </div>
            </>
          )}
        </>
      )}
    </aside>
  );
}
