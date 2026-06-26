import { useEffect, useMemo, useRef, useState } from "react";

import { IcoChevDn, IcoChevUp, IcoGrid, IcoPlus, IcoSend } from "../icons";
import { useChatStore } from "../../stores/useChatStore";
import { useLayoutStore } from "../../stores/useLayoutStore";
import { useResourceStore } from "../../stores/useResourceStore";
import { useTaskStore } from "../../stores/useTaskStore";
import type { DockTab, ResourceItem, ResourceType } from "../../types";

// ── helpers ──────────────────────────────────────────────────────────────────

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

function deriveCategoryCards(resources: ResourceItem[]) {
  const map = new Map<string, { name: string; color: string; count: number }>();
  for (const r of resources) {
    const name = r.categoryName.trim() || "Unsorted";
    const key  = name.toLowerCase();
    if (!map.has(key)) {
      const color = CAT_COLORS[key] ?? "#5B9CF6";
      map.set(key, { name, color, count: 0 });
    }
    map.get(key)!.count++;
  }
  return [...map.values()].sort((a, b) => b.count - a.count);
}

function fmtDate(iso: string): string {
  const d = new Date(iso);
  return isNaN(d.getTime()) ? "" : d.toLocaleDateString("en-US", { month: "short", day: "numeric" });
}

// ── tab config ────────────────────────────────────────────────────────────────

const TABS: Array<{ key: DockTab; label: string }> = [
  { key: "categories", label: "CATEGORIES" },
  { key: "chat",       label: "CHAT"       },
  { key: "tasks",      label: "TASKS"      },
  { key: "library",    label: "LIBRARY"    },
];

// ── Dock ──────────────────────────────────────────────────────────────────────

export default function Dock() {
  const dockOpen   = useLayoutStore((s) => s.dockOpen);
  const dockTab    = useLayoutStore((s) => s.dockTab);
  const setDockOpen = useLayoutStore((s) => s.setDockOpen);
  const openDockTab = useLayoutStore((s) => s.openDockTab);

  const resources      = useResourceStore((s) => s.resources);
  const query          = useResourceStore((s) => s.filters.query);
  const selectResource = useResourceStore((s) => s.selectResource);
  const setRightOpen   = useLayoutStore((s) => s.setRightOpen);

  const messages   = useChatStore((s) => s.messages);
  const isSending  = useChatStore((s) => s.isSending);
  const sendMessage = useChatStore((s) => s.sendMessage);

  const todos     = useTaskStore((s) => s.todos);
  const setStatus = useTaskStore((s) => s.setSelectedTodoStatus);

  const [chatInput, setChatInput] = useState("");
  const threadRef = useRef<HTMLDivElement>(null);

  // auto-scroll chat
  useEffect(() => {
    if (threadRef.current) {
      threadRef.current.scrollTop = threadRef.current.scrollHeight;
    }
  }, [messages]);

  const sendChat = async () => {
    const v = chatInput.trim();
    if (!v) return;
    setChatInput("");
    await sendMessage(v);
  };

  // filtered library list
  const filteredLib = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return resources;
    return resources.filter(
      (r) =>
        r.title.toLowerCase().includes(q) ||
        r.host.toLowerCase().includes(q) ||
        r.categoryName.toLowerCase().includes(q),
    );
  }, [resources, query]);

  const catCards = useMemo(() => deriveCategoryCards(resources), [resources]);
  const maxCatCount = catCards[0]?.count ?? 1;

  const inProgress = todos.filter((t) => t.status === "in_progress");
  const open       = todos.filter((t) => t.status === "open");
  const done       = todos.filter((t) => t.status === "done");

  return (
    <div className={`dock${dockOpen ? " is-open" : ""}`}>
      {/* ── Tab strip ── */}
      <div className="dock-tab-strip">
        <button
          className={`dock-cat-toggle${dockTab === "categories" ? " is-active" : ""}`}
          onClick={() => dockTab === "categories" && dockOpen ? setDockOpen(false) : openDockTab("categories")}
          type="button"
        >
          <IcoGrid />
          <span>CATEGORIES</span>
        </button>

        {TABS.filter((t) => t.key !== "categories").map(({ key, label }) => (
          <button
            key={key}
            className={`dock-tab${dockTab === key && dockOpen ? " is-active" : ""}`}
            onClick={() => dockTab === key && dockOpen ? setDockOpen(false) : openDockTab(key)}
            type="button"
          >
            {label}
          </button>
        ))}

        <div className="dock-spacer" />

        <button
          className="dock-toggle-btn"
          onClick={() => setDockOpen(!dockOpen)}
          type="button"
          aria-label={dockOpen ? "Collapse dock" : "Expand dock"}
        >
          {dockOpen ? <IcoChevDn /> : <IcoChevUp />}
        </button>
      </div>

      {/* ── Bodies ── */}
      {dockOpen && (
        <div className="dock-body">
          {/* categories */}
          {dockTab === "categories" && (
            <div className="dock-categories">
              {catCards.map(({ name, color, count }) => (
                <div key={name} className="cat-card">
                  <div className="cat-card-swatch" style={{ background: color }} />
                  <div className="cat-card-name">{name}</div>
                  <div className="cat-card-count">{count}</div>
                  <div className="cat-card-label">RESOURCES</div>
                  <div className="dot-meter">
                    <div className="dot-meter-track" />
                    <div
                      className="dot-meter-fill"
                      style={{ height: `${Math.round((count / maxCatCount) * 100)}%`, color }}
                    />
                  </div>
                </div>
              ))}
              <div className="cat-card is-new">
                <IcoPlus />
                <span style={{ fontSize: 10, letterSpacing: "0.5px", textTransform: "uppercase" }}>NEW</span>
              </div>
            </div>
          )}

          {/* chat */}
          {dockTab === "chat" && (
            <div className="dock-chat">
              <div className="chat-thread" ref={threadRef}>
                {messages.map((m) => (
                  <div
                    key={m.id}
                    className={`chat-bubble ${m.role === "user" ? "is-user" : "is-ai"}`}
                  >
                    {m.content}
                  </div>
                ))}
              </div>
              <div className="chat-composer">
                <button className="chat-plus-btn" type="button" aria-label="Attach">
                  <IcoPlus />
                </button>
                <input
                  className="chat-composer-input"
                  value={chatInput}
                  onChange={(e) => setChatInput(e.target.value)}
                  onKeyDown={(e) => { if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); void sendChat(); } }}
                  placeholder="Message…"
                  disabled={isSending}
                />
                <button
                  className="chat-send-btn"
                  onClick={() => void sendChat()}
                  disabled={isSending}
                  type="button"
                  aria-label="Send"
                >
                  <IcoSend />
                </button>
              </div>
            </div>
          )}

          {/* tasks */}
          {dockTab === "tasks" && (
            <div className="dock-tasks">
              {[
                { label: "IN PROGRESS", items: inProgress },
                { label: "TO DO",       items: open       },
                { label: "DONE",        items: done       },
              ].map(({ label, items }) => (
                <div key={label} className="task-col">
                  <div className="task-col-header">{label} ({items.length})</div>
                  <div className="task-col-body">
                    {items.map((t) => {
                      const isDone = t.status === "done";
                      return (
                        <div key={t.id} className="task-card-mini">
                          <div className="task-card-mini-row">
                            <button
                              className={`task-checkbox${isDone ? " is-done" : ""}`}
                              onClick={() => void setStatus(isDone ? "open" : "done")}
                              type="button"
                              aria-label={isDone ? "Mark open" : "Mark done"}
                            >
                              {isDone && "✓"}
                            </button>
                            <span className={`task-title-mini${isDone ? " is-done" : ""}`}>{t.title}</span>
                          </div>
                          {t.dueAt && (
                            <div className="task-meta-mini">
                              <span className="task-due-mini">DUE {fmtDate(t.dueAt)}</span>
                            </div>
                          )}
                        </div>
                      );
                    })}
                  </div>
                </div>
              ))}
            </div>
          )}

          {/* library */}
          {dockTab === "library" && (
            <div className="dock-library">
              <div className="lib-header">
                <span className="lib-header-label">RECENT · {filteredLib.length} ITEMS</span>
                <span className="lib-sort">SORT: RECENT ▾</span>
              </div>
              <div className="lib-list">
                {filteredLib.map((r) => {
                  const color = TYPE_COLORS[r.type ?? "link"] ?? "#9A9AA0";
                  return (
                    <div
                      key={r.id}
                      className="lib-row"
                      onClick={() => { selectResource(r.id); setRightOpen(true); }}
                      role="button"
                      tabIndex={0}
                      onKeyDown={(e) => e.key === "Enter" && (selectResource(r.id), setRightOpen(true))}
                    >
                      <span
                        style={{ width: 18, height: 18, background: color, display: "block", flexShrink: 0 }}
                      />
                      <span className="lib-row-title">{r.title || r.url}</span>
                      <span className="lib-row-meta">{r.categoryName} · {fmtDate(r.createdAt)}</span>
                    </div>
                  );
                })}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

// keep named export for old tests
export { Dock };
