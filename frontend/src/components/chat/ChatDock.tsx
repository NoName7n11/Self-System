import { useEffect, useMemo, useRef, useState } from "react";

import { IcoChevDn, IcoChevUp, IcoGrid, IcoPlus, IcoSend } from "../icons";
import { DEMO_CATEGORIES } from "../../lib/demoData";
import { useChatStore } from "../../stores/useChatStore";
import { useLayoutStore } from "../../stores/useLayoutStore";
import { useResourceStore } from "../../stores/useResourceStore";
import { useTaskStore } from "../../stores/useTaskStore";
import type { DockTab, ResourceItem, ResourceType } from "../../types";

const TYPE_COLOR: Record<ResourceType, string> = {
  pdf: "#F67373", link: "#48C78E", note: "#5B9CF6", doc: "#9B59F6", image: "#F6739B",
};
function typeColor(t?: ResourceType): string { return t ? TYPE_COLOR[t] : "#9A9AA0"; }

const TABS: Array<{ key: DockTab; label: string }> = [
  { key: "chat", label: "CHAT" },
  { key: "tasks", label: "TASKS" },
  { key: "library", label: "LIBRARY" },
  { key: "archive", label: "ARCHIVE" },
];

export default function Dock() {
  const dockOpen = useLayoutStore((s) => s.dockOpen);
  const dockTab = useLayoutStore((s) => s.dockTab);
  const openDockTab = useLayoutStore((s) => s.openDockTab);
  const toggleDock = useLayoutStore((s) => s.toggleDock);

  return (
    <div className={`dock${dockOpen ? " is-open" : ""}`}>
      <div className="dock-tab-strip">
        <button className={`dock-cat-toggle${dockTab === "categories" ? " is-active" : ""}`}
          onClick={() => openDockTab("categories")} type="button">
          <IcoGrid /><span>CATEGORIES</span>
        </button>
        <div className="dock-divider" />
        {TABS.map(({ key, label }) => (
          <button key={key} className={`dock-tab${dockTab === key ? " is-active" : ""}`}
            onClick={() => openDockTab(key)} type="button">
            {label}
          </button>
        ))}
        <div className="dock-spacer" />
        <button className="dock-toggle-btn" onClick={toggleDock} type="button" aria-label={dockOpen ? "Collapse" : "Expand"}>
          {dockOpen ? <IcoChevDn /> : <IcoChevUp />}
        </button>
      </div>

      {dockOpen && (
        <div className="dock-body">
          {dockTab === "categories" && <CategoriesTab />}
          {dockTab === "chat" && <ChatTab />}
          {dockTab === "tasks" && <TasksTab />}
          {dockTab === "library" && <LibraryTab />}
          {dockTab === "archive" && <ArchiveTab />}
        </div>
      )}
    </div>
  );
}

// ── categories: ingest area + cat cards ───────────────────────────────────────
function CategoriesTab() {
  const resources = useResourceStore((s) => s.resources);
  const setSelectedCat = useLayoutStore((s) => s.setSelectedCat);
  const [cap, setCap] = useState("");

  const cards = useMemo(() => {
    const counts = new Map<string, number>();
    for (const r of resources) counts.set(r.categoryId, (counts.get(r.categoryId) ?? 0) + 1);
    const max = Math.max(1, ...counts.values());
    return DEMO_CATEGORIES.map((c) => ({
      ...c, count: counts.get(c.id) ?? 0, fillH: Math.round(((counts.get(c.id) ?? 0) / max) * 100),
    }));
  }, [resources]);

  return (
    <div className="dock-categories-wrap">
      <div className="ingest-area">
        <div className="ingest-label">INGEST → AUTO-CLASSIFY INTO GRAPH</div>
        <div className="ingest-bar">
          <input value={cap} onChange={(e) => setCap(e.target.value)} placeholder="PASTE A URL OR TYPE A NOTE…" className="ingest-input" />
          <button className="ingest-attach" type="button" aria-label="Attach"><IcoPlus /></button>
          <button className="ingest-add" type="button">ADD</button>
        </div>
        <div className="ingest-hint">Drop files here, paste a link, or jot a note — it's embedded and routed into the right category automatically.</div>
      </div>

      <div className="cat-list-pane">
        <div className="cat-list-head">
          <span>CATEGORIES</span><span>{cards.length}</span>
        </div>
        <div className="cat-list-body">
          {cards.map((c) => (
            <button key={c.id} className="cat-card-row" onClick={() => setSelectedCat(c.id)} type="button">
              <span className="cat-card-swatch-sm" style={{ background: c.color }} />
              <div className="cat-card-row-main">
                <div className="cat-card-row-top">
                  <span className="cat-card-row-name">{c.name}</span>
                  <span className="cat-card-row-count">{c.count}</span>
                </div>
                <div className="cat-meter">
                  <div className="cat-meter-fill" style={{ width: `${c.fillH}%`, color: c.color }} />
                </div>
              </div>
            </button>
          ))}
          <div className="cat-card-new"><IcoPlus /><span>NEW</span></div>
        </div>
      </div>
    </div>
  );
}

// ── chat ──────────────────────────────────────────────────────────────────────
function ChatTab() {
  const conversations = useChatStore((s) => s.conversations);
  const isSending = useChatStore((s) => s.isSending);
  const sendToConversation = useChatStore((s) => s.sendToConversation);
  const dockConvId = useLayoutStore((s) => s.dockConvId);
  const conv = conversations.find((c) => c.id === dockConvId) ?? conversations[0];
  const messages = conv?.messages ?? [];
  const convTitle = conv?.title ?? "Conversation";
  const [input, setInput] = useState("");
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => { if (ref.current) ref.current.scrollTop = ref.current.scrollHeight; }, [messages]);

  const send = async () => {
    const v = input.trim();
    if (!v || !conv) return;
    setInput("");
    await sendToConversation(conv.id, v);
  };

  return (
    <div className="dock-chat">
      <div className="dock-chat-head">
        <span className="dock-chat-dot" />
        <span className="dock-chat-title">{convTitle}</span>
        <span className="dock-chat-sub">SELECTED CONVERSATION</span>
      </div>
      <div className="chat-thread" ref={ref}>
        {messages.map((m) => (
          <div key={m.id} className="chat-msg-wrap" style={{ alignItems: m.role === "user" ? "flex-end" : "flex-start" }}>
            <div className={`chat-bubble ${m.role === "user" ? "is-user" : "is-ai"}`}>{m.content}</div>
            <span className="chat-role">{m.role === "user" ? "YOU" : "ASSISTANT"}</span>
          </div>
        ))}
      </div>
      <div className="chat-composer">
        <button className="chat-plus-btn" type="button" aria-label="Attach"><IcoPlus /></button>
        <input className="chat-composer-input" value={input} onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => { if (e.key === "Enter") { e.preventDefault(); void send(); } }}
          placeholder="ASK ABOUT YOUR KNOWLEDGE GRAPH…" disabled={isSending} />
        <button className="chat-send-btn" onClick={() => void send()} type="button" aria-label="Send"><IcoSend /></button>
      </div>
    </div>
  );
}

// ── tasks (horizontal columns) ────────────────────────────────────────────────
function TasksTab() {
  const todos = useTaskStore((s) => s.todos);
  const cols: Array<{ key: string; label: string; color: string }> = [
    { key: "in_progress", label: "IN PROGRESS", color: "#F0703C" },
    { key: "open", label: "TO DO", color: "#5B9CF6" },
    { key: "done", label: "DONE", color: "#48C78E" },
  ];
  return (
    <div className="dock-tasks-v2">
      <div className="dock-tasks-head">
        <div style={{ flex: 1 }} />
        <button className="rail-new-chip" type="button"><IcoPlus />NEW TASK</button>
      </div>
      <div className="dock-tasks-cols">
        {cols.map((col) => {
          const items = todos.filter((t) => t.status === col.key);
          return (
            <div key={col.key} className="dock-task-group">
              <div className="dock-task-group-head">
                <span className="task-cat-dot" style={{ background: col.color }} />
                <span className="rail-task-group-label">{col.label}</span>
                <span className="rail-task-group-count">{items.length}</span>
              </div>
              <div className="dock-task-row">
                {items.map((t) => (
                  <div key={t.id} className="dock-task-card">
                    <div className="rail-task-card-top">
                      <span className={`task-checkbox${t.status === "done" ? " is-done" : ""}`}>{t.status === "done" && "✓"}</span>
                      <span className={`rail-task-title${t.status === "done" ? " is-done" : ""}`}>{t.title}</span>
                    </div>
                    <div className="rail-task-card-meta">
                      <span className="task-cat-dot" style={{ background: "#5B9CF6" }} />
                      <span className="task-due-mini">{t.dueAt ? `DUE ${t.dueAt}` : "—"}</span>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

// ── library ───────────────────────────────────────────────────────────────────
function LibraryTab() {
  const resources = useResourceStore((s) => s.resources);
  const query = useResourceStore((s) => s.filters.query);
  const selectResource = useResourceStore((s) => s.selectResource);
  const setRightOpen = useLayoutStore((s) => s.setRightOpen);
  const rows = useMemo(() => {
    const q = query.trim().toLowerCase();
    const visible = resources.filter((r) => !isArchived(r));
    if (!q) return visible;
    return visible.filter((r) => r.title.toLowerCase().includes(q) || r.categoryName.toLowerCase().includes(q));
  }, [resources, query]);

  return (
    <div className="dock-library">
      <div className="lib-header">
        <span className="lib-header-label">RECENT · {rows.length} ITEMS</span>
        <span className="lib-sort">SORT: RECENT ▾</span>
      </div>
      <div className="lib-list">
        {rows.map((r) => (
          <button key={r.id} className="lib-row" onClick={() => { selectResource(r.id); setRightOpen(true); }} type="button">
            <span className="lib-row-badge" style={{ background: typeColor(r.type) }}>{(r.type ?? "link").toUpperCase()}</span>
            <span className="lib-row-title">{r.title || r.url}</span>
            <span className="lib-row-meta">{r.categoryName} · {r.createdAt}</span>
          </button>
        ))}
      </div>
    </div>
  );
}

// ── archive ───────────────────────────────────────────────────────────────────
function ArchiveTab() {
  const resources = useResourceStore((s) => s.resources);
  const items = resources.filter(isArchived);
  return (
    <div className="dock-archive">
      <div className="archive-head">
        <span className="archive-label">ARCHIVE · {items.length}</span>
      </div>
      <div className="archive-list">
        {items.map((r) => (
          <div key={r.id} className="archive-row">
            <span className="archive-badge" style={{ background: typeColor(r.type) }}>{(r.type ?? "link").toUpperCase()}</span>
            <div className="archive-row-main">
              <div className="archive-row-title">{r.title}</div>
              <div className="archive-row-sub">{r.categoryName} · {r.createdAt}</div>
            </div>
            <button className="archive-restore" type="button">RESTORE</button>
          </div>
        ))}
        {items.length === 0 && <div className="archive-empty">NOTHING ARCHIVED</div>}
      </div>
    </div>
  );
}

// archived = category "archive" (demo r20/r21) — backend has an archived flag later
function isArchived(r: ResourceItem): boolean {
  return r.categoryId === "archive";
}
