// Demo / fallback seed dataset — mirrors the design prototype
// (Redesign from scratch_7/Self Systems.dc.html: RES, CATS, tasks, conversations).
// Shown when the backend returns no data, so the UI looks populated like the design.
// Real backend data takes over once resources exist.

import type { ChatMessage, ReminderItem, ResourceItem, TodoItem } from "../types";

export interface DemoCategory {
  id: string;
  name: string;
  color: string;
}

export interface DemoConnection {
  to: string;
  rel: string;
}

export interface DemoResource {
  id: string;
  title: string;
  type: "pdf" | "link" | "note" | "doc" | "image";
  cat: string;
  counter: number;
  date: string;
  host: string;
  tags: string[];
  summary: string;
  connections: DemoConnection[];
  archived?: boolean;
}

export interface DemoConversation {
  id: string;
  title: string;
  messages: Array<{ id: string; role: "user" | "assistant"; content: string; cite?: string; citeRes?: string }>;
}

export const DEMO_CATEGORIES: DemoCategory[] = [
  { id: "research", name: "RESEARCH", color: "#5B9CF6" },
  { id: "ai", name: "AI / ML", color: "#A98BF5" },
  { id: "finance", name: "FINANCE", color: "#48C78E" },
  { id: "people", name: "PEOPLE", color: "#E5B567" },
  { id: "sources", name: "SOURCES", color: "#56B6C2" },
  { id: "archive", name: "ARCHIVE", color: "#E06C75" },
];

// category-to-category weak links (for graph hub edges) [a, b, weight]
export const DEMO_CATLINKS: Array<[string, string, number]> = [
  ["research", "ai", 1],
  ["research", "sources", 0.6],
  ["ai", "finance", 0.45],
  ["ai", "sources", 0.55],
  ["people", "research", 0.4],
  ["finance", "people", 0.3],
  ["archive", "sources", 0.3],
];

export const DEMO_RESOURCES: DemoResource[] = [
  { id: "r1", title: "Raft Consensus Paper", type: "pdf", cat: "research", counter: 5, date: "02 MAR", host: "raft.github.io", tags: ["Distributed", "Consensus"], summary: "A consensus algorithm designed for understandability — equivalent to Paxos in fault-tolerance and performance, but decomposed into leader election, log replication, and safety for easier teaching and implementation.", connections: [{ to: "r4", rel: "cites" }, { to: "r2", rel: "related" }, { to: "r3", rel: "ref by" }] },
  { id: "r2", title: "RAG Tutorial", type: "link", cat: "research", counter: 5, date: "08 MAR", host: "youtube.com", tags: ["RAG", "Retrieval"], summary: "End-to-end walkthrough of retrieval-augmented generation: chunking, embeddings, vector store, and grounded answer synthesis with citations.", connections: [{ to: "r3", rel: "related" }, { to: "r10", rel: "mentioned" }] },
  { id: "r3", title: "Advanced RAG Paper", type: "pdf", cat: "research", counter: 2, date: "26 FEB", host: "arxiv.org", tags: ["RAG"], summary: "Survey of advanced RAG architectures including re-ranking, hybrid search, and query rewriting.", connections: [{ to: "r2", rel: "related" }] },
  { id: "r4", title: "Paxos Made Simple", type: "pdf", cat: "research", counter: 1, date: "15 JAN", host: "lamport.org", tags: ["Consensus"], summary: "Lamport's accessible restatement of the Paxos consensus protocol.", connections: [{ to: "r1", rel: "cited by" }] },
  { id: "r5", title: "Attention Is All You Need", type: "pdf", cat: "ai", counter: 4, date: "10 FEB", host: "arxiv.org", tags: ["Transformers"], summary: "Introduces the Transformer architecture built entirely on self-attention, removing recurrence and convolution.", connections: [{ to: "r6", rel: "related" }, { to: "r7", rel: "ref by" }] },
  { id: "r6", title: "ML Survey 2026", type: "pdf", cat: "ai", counter: 3, date: "01 MAR", host: "arxiv.org", tags: ["Survey"], summary: "Broad survey of the 2026 machine-learning landscape across modalities and training regimes.", connections: [{ to: "r5", rel: "related" }] },
  { id: "r7", title: "Transformers from Scratch", type: "link", cat: "ai", counter: 2, date: "20 FEB", host: "github.io", tags: ["Tutorial"], summary: "Builds a Transformer from first principles in annotated code.", connections: [{ to: "r5", rel: "cites" }] },
  { id: "r8", title: "GBUS Design Notes", type: "note", cat: "ai", counter: 1, date: "10 MAR", host: "local", tags: ["Internal"], summary: "Working notes on the Generalized Behavioral Understanding System — weighted signals and multi-interest modeling.", connections: [] },
  { id: "r9", title: "AI in Healthcare", type: "link", cat: "ai", counter: 2, date: "14 FEB", host: "nature.com", tags: ["Healthcare"], summary: "Review of clinical applications of machine learning and their deployment challenges.", connections: [{ to: "r6", rel: "mentioned" }] },
  { id: "r10", title: "RAG in Trading", type: "link", cat: "finance", counter: 1, date: "18 FEB", host: "medium.com", tags: ["RAG", "Finance"], summary: "Applies retrieval-augmented generation to financial research workflows.", connections: [{ to: "r2", rel: "cites" }] },
  { id: "r11", title: "Q3 Roadmap", type: "doc", cat: "finance", counter: 3, date: "05 MAR", host: "local", tags: ["Planning"], summary: "Quarterly roadmap covering sync server hardening and mobile companion scoping.", connections: [] },
  { id: "r12", title: "Startup Funding Guide", type: "pdf", cat: "finance", counter: 2, date: "28 JAN", host: "yc.com", tags: ["Funding"], summary: "Practical guide to early-stage fundraising stages and terms.", connections: [] },
  { id: "r13", title: "Coinbase Due Diligence", type: "note", cat: "finance", counter: 1, date: "12 MAR", host: "local", tags: ["Research"], summary: "Diligence notes on holdings, cost basis, and dividend yield.", connections: [] },
  { id: "r14", title: "Alex — Distributed Sys", type: "note", cat: "people", counter: 1, date: "03 MAR", host: "local", tags: ["Contact"], summary: "Collaborator on consensus and replication topics.", connections: [{ to: "r1", rel: "mentioned" }] },
  { id: "r15", title: "Paco — Design", type: "note", cat: "people", counter: 1, date: "06 MAR", host: "local", tags: ["Contact"], summary: "Design partner; owns the inspector and dock visual language.", connections: [] },
  { id: "r16", title: "Quinn — Eng Lead", type: "note", cat: "people", counter: 1, date: "09 MAR", host: "local", tags: ["Contact"], summary: "Engineering lead tracking the graph and sync workstreams.", connections: [] },
  { id: "r17", title: "arXiv Daily Feed", type: "link", cat: "sources", counter: 2, date: "11 MAR", host: "arxiv.org", tags: ["Feed"], summary: "Subscribed feed surfacing new preprints matching your interest profile.", connections: [{ to: "r5", rel: "source of" }, { to: "r6", rel: "source of" }] },
  { id: "r18", title: "Hacker News", type: "link", cat: "sources", counter: 1, date: "11 MAR", host: "news.yc", tags: ["Feed"], summary: "Aggregator feed for technology discussion and launches.", connections: [] },
  { id: "r19", title: "Hackathon 2026", type: "link", cat: "sources", counter: 2, date: "04 MAR", host: "devpost.com", tags: ["Event"], summary: "AI Innovation hackathon — deadline detected, tasks auto-generated.", connections: [{ to: "r17", rel: "found via" }] },
  { id: "r20", title: "Old Hackathon 2024", type: "link", cat: "archive", counter: 1, date: "02 NOV", host: "devpost.com", tags: ["Expired"], summary: "Archived: event date passed.", connections: [], archived: true },
  { id: "r21", title: "Expired Internship", type: "link", cat: "archive", counter: 1, date: "01 JUN", host: "lever.co", tags: ["Expired"], summary: "Archived: deadline exceeded.", connections: [], archived: true },
];

export const DEMO_RECENT_IDS = ["r2", "r1", "r8", "r19", "r11"];

export const DEMO_CONVERSATIONS: DemoConversation[] = [
  { id: "cv1", title: "Raft consensus — Q&A", messages: [
    { id: "c1", role: "user", content: "How does leader election work in Raft?" },
    { id: "c2", role: "assistant", content: "When a follower's election timeout elapses with no heartbeat, it becomes a candidate, increments its term, votes for itself, and requests votes from peers. A candidate that wins a majority becomes leader and starts sending heartbeats.", cite: "Raft Consensus Paper §5.2", citeRes: "r1" },
    { id: "c3", role: "user", content: "What guarantees safety across terms?" },
    { id: "c4", role: "assistant", content: "The Leader Completeness property: any entry committed in a given term is present in the logs of all higher-term leaders, enforced by the election restriction on up-to-date logs.", cite: "Raft Consensus Paper §5.4", citeRes: "r1" },
  ] },
  { id: "cv2", title: "RAG pipeline design", messages: [
    { id: "c5", role: "user", content: "What's the cleanest chunking strategy for the RAG tutorial?" },
    { id: "c6", role: "assistant", content: "Start with semantic chunking around 512 tokens with ~10% overlap; the tutorial benchmarks this against fixed windows and it wins on recall.", cite: "RAG Tutorial", citeRes: "r2" },
  ] },
  { id: "cv3", title: "GBUS weighted signals", messages: [
    { id: "c7", role: "user", content: "How should deletions affect the interest profile?" },
    { id: "c8", role: "assistant", content: "Treat deletion as an ambiguous signal (weight 0.1) — it can mean \"consumed\" rather than \"uninterested,\" so it barely moves the profile.", cite: "GBUS Design Notes", citeRes: "r8" },
  ] },
];

export interface DemoTask {
  id: string;
  title: string;
  due: string;
  status: "open" | "in_progress" | "done";
  cat: string;
}

export const DEMO_TASKS: DemoTask[] = [
  { id: "t1", title: "Review hackathon requirements", due: "12 JAN", status: "done", cat: "sources" },
  { id: "t2", title: "Prepare project proposal", due: "18 JAN", status: "in_progress", cat: "sources" },
  { id: "t3", title: "Submit application", due: "20 JAN", status: "open", cat: "sources" },
  { id: "t4", title: "Read Raft paper §5 (safety)", due: "15 MAR", status: "open", cat: "research" },
  { id: "t5", title: "Summarize RAG tutorial", due: "10 MAR", status: "done", cat: "research" },
  { id: "t6", title: "Draft GBUS weighted-signal spec", due: "14 MAR", status: "in_progress", cat: "ai" },
];

export interface DemoNotif {
  id: string;
  icon: string;
  color: string;
  title: string;
  body: string;
  time: string;
}

export const DEMO_NOTIFS: DemoNotif[] = [
  { id: "n1", icon: "plus", color: "#48C78E", title: "New resource added", body: "“RAG in Trading” was captured from medium.com", time: "2m" },
  { id: "n2", icon: "gear", color: "#5B9CF6", title: "App update available", body: "Self Systems v0.1.1 — graph perf + map view", time: "1h" },
  { id: "n3", icon: "tasks", color: "#F0703C", title: "Deadline detected", body: "Hackathon 2026 — 3 tasks auto-generated", time: "4h" },
  { id: "n4", icon: "archive", color: "#E5B567", title: "2 resources archived", body: "Expired events moved to ARCHIVE", time: "1d" },
];

const CAT_NAME_BY_ID: Record<string, string> = Object.fromEntries(
  DEMO_CATEGORIES.map((c) => [c.id, c.name]),
);

// ── Adapters: demo shapes → app store shapes ──────────────────────────────────

export function demoResourcesAsItems(): ResourceItem[] {
  return DEMO_RESOURCES.map((r) => ({
    id: r.id,
    url: r.host === "local" ? "" : `https://${r.host}`,
    host: r.host,
    title: r.title,
    summary: r.summary,
    categoryId: r.cat,
    categoryName: CAT_NAME_BY_ID[r.cat] ?? r.cat,
    userOverride: false,
    type: r.type,
    createdAt: r.date,
    updatedAt: r.date,
  }));
}

export function demoTasksAsTodos(): TodoItem[] {
  return DEMO_TASKS.map((t) => ({
    id: t.id,
    title: t.title,
    details: "",
    status: t.status,
    dueAt: t.due,
    resourceId: "",
    cat: t.cat,
    createdAt: "",
    updatedAt: "",
  }));
}

export function demoRemindersAsItems(): ReminderItem[] {
  return [];
}

export function demoChatMessages(): ChatMessage[] {
  // flatten the default (cv1) conversation into the chat store shape
  return DEMO_CONVERSATIONS[0].messages.map((m) => ({
    id: m.id,
    role: m.role,
    content: m.content,
    createdAt: "",
  }));
}
