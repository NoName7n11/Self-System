import { useMemo, useState } from "react";

import { IcoChevL, IcoChevR, IcoTrend } from "../icons";
import { DEMO_CATEGORIES, DEMO_RESOURCES } from "../../lib/demoData";
import { useChatStore } from "../../stores/useChatStore";
import { useLayoutStore } from "../../stores/useLayoutStore";
import { useResourceStore } from "../../stores/useResourceStore";
import type { ResourceType } from "../../types";

const TYPE_COLOR: Record<ResourceType, string> = {
  pdf: "#F67373", link: "#48C78E", note: "#5B9CF6", doc: "#9B59F6", image: "#F6739B",
};
const CAT_COLOR: Record<string, string> = Object.fromEntries(DEMO_CATEGORIES.map((c) => [c.id, c.color]));
const DEMO_BY_ID = Object.fromEntries(DEMO_RESOURCES.map((r) => [r.id, r]));

const SUGGESTED_QS = [
  "How does leader election work here?",
  "Key differences from related work?",
  "Which open questions need follow-up?",
];

// kept exported for any remaining importers
export function getResourceFormCopy() { return { heading: "Inspector", subheading: "" }; }

export default function Inspector() {
  const rightOpen = useLayoutStore((s) => s.rightOpen);
  const toggleRight = useLayoutStore((s) => s.toggleRight);
  const openDockTab = useLayoutStore((s) => s.openDockTab);

  const resources = useResourceStore((s) => s.resources);
  const selectedId = useResourceStore((s) => s.selectedResourceId);
  const selectResource = useResourceStore((s) => s.selectResource);
  const deleteSelectedResource = useResourceStore((s) => s.deleteSelectedResource);
  const sendMessage = useChatStore((s) => s.sendMessage);

  const [ask, setAsk] = useState("");

  const resource = useMemo(() => resources.find((r) => r.id === selectedId) ?? null, [resources, selectedId]);
  const demo = resource ? DEMO_BY_ID[resource.id] : undefined;
  const typeCol = resource ? (TYPE_COLOR[resource.type ?? "link"] ?? "#9A9AA0") : "#9A9AA0";
  const catCol = resource ? (CAT_COLOR[resource.categoryId] ?? "#5B9CF6") : "#5B9CF6";

  const connections = (demo?.connections ?? []).map((c) => {
    const target = resources.find((r) => r.id === c.to);
    return { id: c.to, title: target?.title ?? c.to, rel: c.rel.toUpperCase(), color: target ? (CAT_COLOR[target.categoryId] ?? "#5B9CF6") : "#5B9CF6" };
  });
  const tags = demo?.tags ?? [];
  const counter = demo?.counter ?? 1;

  const sendAsk = async (q: string) => {
    const v = q.trim();
    if (!v) return;
    setAsk("");
    await sendMessage(resource ? `[${resource.title}] ${v}` : v);
    openDockTab("chat");
  };

  return (
    <aside className={`right-rail${rightOpen ? " is-open" : ""}`}>
      {!rightOpen && (
        <div className="right-rail-collapsed">
          <button className="right-expand-btn" onClick={toggleRight} type="button" aria-label="Open inspector"><IcoChevL /></button>
          {resource && <span className="right-cat-swatch" style={{ background: catCol }} />}
        </div>
      )}

      {rightOpen && (
        <>
          <div className="inspector-header">
            <span className="inspector-title">INSPECTOR</span>
            <button className="inspector-collapse-btn" onClick={toggleRight} type="button" aria-label="Close"><IcoChevR /></button>
          </div>

          {!resource ? (
            <div className="inspector-empty-state">Select a node on the graph to inspect it.</div>
          ) : (
            <>
              <div className="inspector-body">
                <div className="inspector-preview">
                  <div className="inspector-preview-host">{resource.host || "—"}</div>
                  <div className="inspector-preview-type" style={{ color: typeCol }}>{(resource.type ?? "link").toUpperCase()}</div>
                  <div className="inspector-preview-cat" style={{ background: catCol }} />
                </div>

                <div className="inspector-section">
                  <div className="inspector-resource-title">{resource.title || resource.url || "Untitled"}</div>
                  <div className="inspector-meta-row">
                    <span className="type-badge-solid" style={{ background: typeCol }}>{(resource.type ?? "link").toUpperCase()}</span>
                    <span className="inspector-date">ADDED {resource.createdAt}</span>
                    <div style={{ flex: 1 }} />
                    <span className="inspector-counter"><span className="inspector-counter-dot" />×{counter}</span>
                  </div>
                  <div className="inspector-tags">
                    <span className="tag-chip">{resource.categoryName}</span>
                    {tags.map((t) => <span key={t} className="tag-chip">{t}</span>)}
                  </div>
                </div>

                <div className="inspector-section">
                  <div className="inspector-actions">
                    <button className="inspector-action-btn" onClick={() => resource.url && window.open(resource.url, "_blank")} type="button">OPEN</button>
                    <button className="inspector-action-btn" type="button">EDIT</button>
                    <button className="inspector-action-btn" type="button">ARCHIVE</button>
                    <button className="inspector-action-btn is-danger" onClick={() => void deleteSelectedResource()} type="button">DELETE</button>
                  </div>
                </div>

                <div className="inspector-section">
                  <div className="inspector-section-label">CONNECTIONS · {connections.length}</div>
                  {connections.length === 0 ? (
                    <div className="inspector-empty">No connections yet.</div>
                  ) : (
                    <div className="inspector-conn-list">
                      {connections.map((c) => (
                        <button key={c.id + c.rel} className="inspector-conn-row" onClick={() => selectResource(c.id)} type="button">
                          <span className="inspector-conn-dot" style={{ background: c.color }} />
                          <span className="inspector-conn-title">{c.title}</span>
                          <span className="inspector-conn-rel">{c.rel}</span>
                        </button>
                      ))}
                    </div>
                  )}
                </div>

                {resource.summary && (
                  <div className="inspector-section" style={{ borderBottom: "none" }}>
                    <div className="inspector-section-label">AI SUMMARY</div>
                    <div className="inspector-summary-text">{resource.summary}</div>
                    {SUGGESTED_QS.map((q) => (
                      <div key={q} className="suggested-q" onClick={() => void sendAsk(q)} role="button" tabIndex={0}
                        onKeyDown={(e) => e.key === "Enter" && void sendAsk(q)}>
                        <span className="suggested-q-arrow">›</span>{q}
                      </div>
                    ))}
                  </div>
                )}
              </div>

              <div className="inspector-footer">
                <input className="inspector-ask-input" value={ask} onChange={(e) => setAsk(e.target.value)}
                  onKeyDown={(e) => { if (e.key === "Enter") void sendAsk(ask); }} placeholder="ASK ABOUT THIS RESOURCE…" />
                <button className="inspector-send-btn" onClick={() => void sendAsk(ask)} type="button" aria-label="Send"><IcoTrend /></button>
              </div>
            </>
          )}
        </>
      )}
    </aside>
  );
}
