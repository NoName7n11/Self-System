import { useEffect, useMemo, useRef, useState } from "react";

import {
  IcoChevDn, IcoChevL, IcoChevR, IcoChat, IcoDots, IcoGear, IcoLibrary,
  IcoLogo, IcoPlus, IcoSearch, IcoSend, IcoTasks, IcoTrend,
} from "../icons";
import { DEMO_CATEGORIES, DEMO_RECENT_IDS } from "../../lib/demoData";
import { useChatStore } from "../../stores/useChatStore";
import { useLayoutStore } from "../../stores/useLayoutStore";
import { useResourceStore } from "../../stores/useResourceStore";
import { useTaskStore } from "../../stores/useTaskStore";
import type { DockTab, LeftView, ResourceItem, ResourceType } from "../../types";

const TYPE_COLOR: Record<ResourceType, string> = {
  pdf: "#F67373", link: "#48C78E", note: "#5B9CF6", doc: "#9B59F6", image: "#F6739B",
};
const CAT_COLOR: Record<string, string> = Object.fromEntries(
  DEMO_CATEGORIES.map((c) => [c.id, c.color]),
);

function typeColor(t?: ResourceType): string { return t ? TYPE_COLOR[t] : "#9A9AA0"; }

// kept exported for any remaining importers
export function deriveFavorites(resources: ResourceItem[], limit = 3): Array<[string, number]> {
  const m = new Map<string, number>();
  for (const r of resources) m.set(r.categoryName.trim() || "Unsorted", (m.get(r.categoryName.trim() || "Unsorted") ?? 0) + 1);
  return [...m.entries()].sort((a, b) => b[1] - a[1]).slice(0, limit);
}
export function deriveRecents(resources: ResourceItem[], limit = 5): ResourceItem[] {
  return [...resources].sort((a, b) => b.createdAt.localeCompare(a.createdAt)).slice(0, limit);
}

const NAV: Array<{ view: LeftView; tab: DockTab; label: string; Icon: () => React.ReactElement }> = [
  { view: "chat", tab: "chat", label: "CHAT", Icon: IcoChat },
  { view: "tasks", tab: "tasks", label: "TASKS", Icon: IcoTasks },
  { view: "library", tab: "library", label: "LIBRARY", Icon: IcoLibrary },
];

const LIB_CHIPS = ["all", "pdf", "link", "note"];

export default function Sidebar() {
  const leftCollapsed = useLayoutStore((s) => s.leftCollapsed);
  const leftView = useLayoutStore((s) => s.leftView);
  const recentOpen = useLayoutStore((s) => s.recentOpen);
  const catsOpen = useLayoutStore((s) => s.catsOpen);
  const selectedCat = useLayoutStore((s) => s.selectedCat);
  const libFilter = useLayoutStore((s) => s.libFilter);
  const toggleLeft = useLayoutStore((s) => s.toggleLeft);
  const setLeftView = useLayoutStore((s) => s.setLeftView);
  const openDockTab = useLayoutStore((s) => s.openDockTab);
  const toggleRecent = useLayoutStore((s) => s.toggleRecent);
  const toggleCats = useLayoutStore((s) => s.toggleCats);
  const setLibFilter = useLayoutStore((s) => s.setLibFilter);
  const toggleNotif = useLayoutStore((s) => s.toggleNotif);
  const setRightOpen = useLayoutStore((s) => s.setRightOpen);

  const resources = useResourceStore((s) => s.resources);
  const query = useResourceStore((s) => s.filters.query);
  const setQuery = useResourceStore((s) => s.setQuery);
  const selectResource = useResourceStore((s) => s.selectResource);

  const byId = useMemo(() => new Map(resources.map((r) => [r.id, r])), [resources]);
  const recents = useMemo(() => {
    const seeded = DEMO_RECENT_IDS.map((id) => byId.get(id)).filter(Boolean) as ResourceItem[];
    return seeded.length > 0 ? seeded : deriveRecents(resources);
  }, [byId, resources]);

  const select = (id: string) => { selectResource(id); setRightOpen(true); };

  return (
    <aside className={`left-rail${leftCollapsed ? " is-collapsed" : ""}`}>
      {/* header */}
      <div className="rail-header">
        <div className="logo-chip" onClick={() => leftCollapsed && setLeftView("home")}>
          <IcoLogo />
        </div>
        {!leftCollapsed && (
          <>
            <div className="rail-wordmark">
              <div className="rail-wordmark-name">SELF SYSTEMS</div>
              <div className="rail-wordmark-sub">LOCAL · v0.1.0</div>
            </div>
            <button className="rail-collapse-btn" onClick={toggleLeft} type="button" aria-label="Collapse">
              <IcoChevL />
            </button>
          </>
        )}
      </div>

      {/* collapsed strip */}
      {leftCollapsed && (
        <div className="rail-collapsed-body">
          {NAV.map(({ view, label, Icon }) => (
            <button key={view} className="rail-collapsed-btn" onClick={() => setLeftView(view)} type="button" aria-label={label}>
              <Icon />
            </button>
          ))}
          <div style={{ flex: 1 }} />
          <button className="rail-collapsed-btn" type="button" aria-label="Settings"><IcoGear /></button>
        </div>
      )}

      {/* ===== HOME ===== */}
      {!leftCollapsed && leftView === "home" && (
        <div className="rail-body">
          <div className="rail-search">
            <IcoSearch />
            <input value={query} onChange={(e) => setQuery(e.target.value)} placeholder="SEARCH RESOURCES, TAGS…" aria-label="Search" />
            <span className="rail-search-hint">/</span>
          </div>

          <nav className="rail-nav">
            {NAV.map(({ view, tab, label, Icon }) => (
              <button key={view} className="nav-row" onClick={() => setLeftView(view)} type="button">
                <span className="nav-row-icon"><Icon /></span>
                <span className="nav-row-label">{label}</span>
                <span className="nav-affordance" role="button" aria-label={`Open ${label} in dock`}
                  onClick={(e) => { e.stopPropagation(); openDockTab(tab); }}>
                  <IcoTrend />
                </span>
              </button>
            ))}
          </nav>

          <div className="rail-rule" />

          {/* recent */}
          <button className="rail-section-toggle" onClick={toggleRecent} type="button">
            <span className="rail-chev">{recentOpen ? <IcoChevDn /> : <IcoChevR />}</span>
            <span className="rail-section-label-inline">RECENT</span>
            <span className="rail-section-count">{recents.length}</span>
          </button>
          {recentOpen && (
            <div className="rail-list">
              {recents.map((r) => (
                <button key={r.id} className="rail-recent-row" onClick={() => select(r.id)} type="button">
                  <span className="rail-type-swatch" style={{ background: typeColor(r.type) }} />
                  <span className="rail-recent-title">{r.title || r.host || r.url}</span>
                  <span className="rail-recent-type">{(r.type ?? "link").toUpperCase()}</span>
                </button>
              ))}
              {recents.length === 0 && <div className="rail-empty">NO RECENT ITEMS</div>}
            </div>
          )}

          <div className="rail-rule" />

          {/* category nodes */}
          <button className="rail-section-toggle" onClick={toggleCats} type="button">
            <span className="rail-chev">{catsOpen ? <IcoChevDn /> : <IcoChevR />}</span>
            <span className="rail-section-label-inline">CATEGORY NODES</span>
            {selectedCat && (
              <span className="rail-selcat">
                <span className="rail-selcat-dot" style={{ background: CAT_COLOR[selectedCat] ?? "#5B9CF6" }} />
                <span className="rail-selcat-name">{DEMO_CATEGORIES.find((c) => c.id === selectedCat)?.name}</span>
              </span>
            )}
          </button>
          {catsOpen && (
            selectedCat ? (
              <>
                <div className="rail-catnodes-head">
                  <span>{resources.filter((r) => r.categoryId === selectedCat).length} NODES</span>
                  <span className="rail-viewall" onClick={() => openDockTab("categories")}>VIEW ALL →</span>
                </div>
                <div className="rail-list">
                  {resources.filter((r) => r.categoryId === selectedCat).map((r) => (
                    <button key={r.id} className="rail-catnode-row" onClick={() => select(r.id)} type="button">
                      <span className="rail-catnode-badge" style={{ background: typeColor(r.type) }}>{(r.type ?? "link").toUpperCase()}</span>
                      <span className="rail-recent-title">{r.title}</span>
                    </button>
                  ))}
                </div>
              </>
            ) : (
              <div className="rail-cathint">
                Select a category in the graph or dock to see its nodes here.
                <span className="rail-viewall" onClick={() => openDockTab("categories")}>BROWSE CATEGORIES →</span>
              </div>
            )
          )}

          <div style={{ flex: 1 }} />

          {/* update banner */}
          <button className="rail-update" onClick={toggleNotif} type="button">
            <span className="rail-update-dot" />
            <span className="rail-update-label">UPDATE AVAILABLE · v0.1.1</span>
            <span className="rail-update-chev"><IcoChevR /></span>
          </button>

          {/* footer */}
          <div className="rail-footer">
            <div className="rail-avatar">N</div>
            <div className="rail-user">
              <div className="rail-user-name">noname</div>
              <div className="rail-user-sub">local · single user</div>
            </div>
            <button className="rail-gear-btn" type="button" aria-label="Settings"><IcoGear /></button>
          </div>
        </div>
      )}

      {/* ===== CHAT / TASKS / LIBRARY ===== */}
      {!leftCollapsed && leftView === "chat" && <RailChat />}
      {!leftCollapsed && leftView === "tasks" && <RailTasks />}
      {!leftCollapsed && leftView === "library" && <RailLibrary filter={libFilter} setFilter={setLibFilter} onSelect={select} />}
    </aside>
  );
}

// ── rail chat (conversation list → thread) ────────────────────────────────────
function RailChat() {
  const setLeftView = useLayoutStore((s) => s.setLeftView);
  const openConvInDock = useLayoutStore((s) => s.openConvInDock);
  const setDockConvId = useLayoutStore((s) => s.setDockConvId);
  const dockConvId = useLayoutStore((s) => s.dockConvId);

  const conversations = useChatStore((s) => s.conversations);
  const newConversation = useChatStore((s) => s.newConversation);
  const renameConversation = useChatStore((s) => s.renameConversation);
  const archiveConversation = useChatStore((s) => s.archiveConversation);
  const deleteConversation = useChatStore((s) => s.deleteConversation);
  const sendToConversation = useChatStore((s) => s.sendToConversation);

  const [convId, setConvId] = useState<string | null>(null);
  const [menuId, setMenuId] = useState<string | null>(null);
  const [renameId, setRenameId] = useState<string | null>(null);
  const [renameVal, setRenameVal] = useState("");
  const [input, setInput] = useState("");
  const threadRef = useRef<HTMLDivElement>(null);

  const visible = conversations.filter((c) => !c.archived);
  const conv = conversations.find((c) => c.id === convId) ?? null;

  useEffect(() => {
    if (threadRef.current) threadRef.current.scrollTop = threadRef.current.scrollHeight;
  }, [conv?.messages.length]);

  const newChat = () => { const id = newConversation(); setConvId(id); };
  const commitRename = () => { if (renameId) renameConversation(renameId, renameVal); setRenameId(null); };
  const doArchive = (id: string) => { archiveConversation(id); setMenuId(null); if (convId === id) setConvId(null); };
  const doDelete = (id: string) => {
    const fallback = deleteConversation(id);
    setMenuId(null);
    if (convId === id) setConvId(null);
    if (dockConvId === id && fallback) setDockConvId(fallback);
  };
  const send = async () => {
    const v = input.trim();
    if (!v || !conv) return;
    setInput("");
    await sendToConversation(conv.id, v);
  };

  // ── conversation list ──
  if (!conv) {
    return (
      <div className="rail-view">
        <div className="rail-view-head">
          <button className="rail-back" onClick={() => setLeftView("home")} type="button"><IcoChevL /></button>
          <span className="rail-view-title">CHATS</span>
          <button className="rail-new-chip" onClick={newChat} type="button"><IcoPlus />NEW CHAT</button>
        </div>
        <div className="rail-conv-list">
          {visible.map((c) => (
            <div key={c.id} className="rail-conv-row" onClick={() => renameId !== c.id && setConvId(c.id)}>
              {/* dot is accent only for the conversation open in the dock, else grey */}
              <span className="rail-conv-dot" style={{ background: c.id === dockConvId ? "#F0703C" : "#3C3C44" }} />
              <div className="rail-conv-main">
                {renameId === c.id ? (
                  <input
                    className="rail-conv-rename"
                    value={renameVal}
                    autoFocus
                    onClick={(e) => e.stopPropagation()}
                    onChange={(e) => setRenameVal(e.target.value)}
                    onKeyDown={(e) => { if (e.key === "Enter") commitRename(); else if (e.key === "Escape") setRenameId(null); }}
                    onBlur={commitRename}
                  />
                ) : (
                  <>
                    <div className="rail-conv-title">{c.title}</div>
                    <div className="rail-conv-preview">{c.messages[c.messages.length - 1]?.content ?? "No messages yet"}</div>
                  </>
                )}
              </div>
              <span className="rail-conv-dots" role="button" aria-label="Options"
                onClick={(e) => { e.stopPropagation(); setMenuId(menuId === c.id ? null : c.id); }}>
                <IcoDots />
              </span>

              {menuId === c.id && (
                <>
                  <div className="ctx-scrim" onClick={(e) => { e.stopPropagation(); setMenuId(null); }} />
                  <div className="ctx-menu" onClick={(e) => e.stopPropagation()}>
                    <button className="ctx-item" onClick={() => { setRenameId(c.id); setRenameVal(c.title); setMenuId(null); }} type="button">Edit name</button>
                    <button className="ctx-item" onClick={() => { openConvInDock(c.id); setMenuId(null); }} type="button">Open in dock</button>
                    <button className="ctx-item" onClick={() => doArchive(c.id)} type="button">Archive</button>
                    <div className="ctx-divider" />
                    <button className="ctx-item is-danger" onClick={() => doDelete(c.id)} type="button">Delete</button>
                  </div>
                </>
              )}
            </div>
          ))}
          {visible.length === 0 && <div className="rail-empty">NO CONVERSATIONS</div>}
        </div>
      </div>
    );
  }

  // ── conversation thread ──
  return (
    <div className="rail-view">
      <div className="rail-view-head">
        <button className="rail-back" onClick={() => setConvId(null)} type="button"><IcoChevL /></button>
        <span className="rail-view-title ellipsis">{conv.title}</span>
        <button className="rail-icon-btn" onClick={() => openConvInDock(conv.id)} type="button" aria-label="Open in dock"><IcoTrend /></button>
      </div>
      <div className="rail-chat-thread" ref={threadRef}>
        {conv.messages.map((m) => (
          <div key={m.id} className={`chat-bubble ${m.role === "user" ? "is-user" : "is-ai"}`}>
            {m.content}
          </div>
        ))}
        {conv.messages.length === 0 && <div className="rail-empty">Start the conversation below.</div>}
      </div>
      <div className="rail-composer">
        <button className="chat-plus-btn" type="button"><IcoPlus /></button>
        <input className="chat-composer-input" value={input} onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => { if (e.key === "Enter") { e.preventDefault(); void send(); } }} placeholder="MESSAGE…" />
        <button className="chat-send-btn" onClick={() => void send()} type="button" aria-label="Send"><IcoSend /></button>
      </div>
    </div>
  );
}

// ── rail tasks (grouped columns) ──────────────────────────────────────────────
function RailTasks() {
  const setLeftView = useLayoutStore((s) => s.setLeftView);
  const todos = useTaskStore((s) => s.todos);
  const toggleTodo = useTaskStore((s) => s.toggleTodo);
  const quickAddTask = useTaskStore((s) => s.quickAddTask);
  const cols: Array<{ key: string; label: string; color: string }> = [
    { key: "in_progress", label: "IN PROGRESS", color: "#F0703C" },
    { key: "open", label: "TO DO", color: "#5B9CF6" },
    { key: "done", label: "DONE", color: "#48C78E" },
  ];
  return (
    <div className="rail-view">
      <div className="rail-view-head">
        <button className="rail-back" onClick={() => setLeftView("home")} type="button"><IcoChevL /></button>
        <span className="rail-view-title">TASKS</span>
        <button className="rail-new-chip" onClick={() => quickAddTask()} type="button"><IcoPlus />NEW</button>
      </div>
      <div className="rail-tasks-body">
        {cols.map((col) => {
          const items = todos.filter((t) => t.status === col.key);
          return (
            <div key={col.key} className="rail-task-group">
              <div className="rail-task-group-head">
                <span className="task-cat-dot" style={{ background: col.color }} />
                <span className="rail-task-group-label">{col.label}</span>
                <span className="rail-task-group-count">{items.length}</span>
              </div>
              {items.map((t) => (
                <div key={t.id} className="rail-task-card">
                  <div className="rail-task-card-top">
                    <button
                      className={`task-checkbox${t.status === "done" ? " is-done" : ""}`}
                      onClick={() => toggleTodo(t.id)}
                      type="button"
                      aria-label={t.status === "done" ? "Mark open" : "Mark done"}
                    >
                      {t.status === "done" && "✓"}
                    </button>
                    <span className={`rail-task-title${t.status === "done" ? " is-done" : ""}`}>{t.title}</span>
                  </div>
                  <div className="rail-task-card-meta">
                    <span className="task-cat-dot" style={{ background: CAT_COLOR[t.cat ?? ""] ?? "#5B9CF6" }} />
                    <span className="task-due-mini">DUE {t.dueAt || "—"}</span>
                  </div>
                </div>
              ))}
            </div>
          );
        })}
      </div>
    </div>
  );
}

// ── rail library (filter chips + rows) ────────────────────────────────────────
function RailLibrary({ filter, setFilter, onSelect }: { filter: string; setFilter: (f: string) => void; onSelect: (id: string) => void }) {
  const setLeftView = useLayoutStore((s) => s.setLeftView);
  const openDockTab = useLayoutStore((s) => s.openDockTab);
  const resources = useResourceStore((s) => s.resources);
  const rows = resources.filter((r) => filter === "all" || r.type === filter);
  return (
    <div className="rail-view">
      <div className="rail-view-head">
        <button className="rail-back" onClick={() => setLeftView("home")} type="button"><IcoChevL /></button>
        <span className="rail-view-title">LIBRARY</span>
        <button className="rail-icon-btn" onClick={() => openDockTab("categories")} type="button" aria-label="Open in dock"><IcoTrend /></button>
      </div>
      <div className="rail-lib-chips">
        {LIB_CHIPS.map((c) => (
          <button key={c} className={`rail-lib-chip${filter === c ? " is-active" : ""}`} onClick={() => setFilter(c)} type="button">
            {c.toUpperCase()}
          </button>
        ))}
      </div>
      <div className="rail-lib-sort">
        <span>{rows.length} ITEMS</span>
        <span>SORT: RECENT ▾</span>
      </div>
      <div className="rail-list rail-lib-list">
        {rows.map((r) => (
          <button key={r.id} className="rail-lib-row" onClick={() => onSelect(r.id)} type="button">
            <span className="rail-lib-badge" style={{ background: typeColor(r.type) }}>{(r.type ?? "link").toUpperCase()}</span>
            <span className="rail-recent-title">{r.title}</span>
            <span className="rail-lib-date">{r.createdAt}</span>
          </button>
        ))}
      </div>
    </div>
  );
}
