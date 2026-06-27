import { create } from "zustand";

import { sendChatCommand } from "../api/client";
import { DEMO_CONVERSATIONS } from "../lib/demoData";
import type { ChatMessage } from "../types";
import { useResourceStore } from "./useResourceStore";
import { useTaskStore } from "./useTaskStore";

export interface Conversation {
  id: string;
  title: string;
  archived: boolean;
  messages: ChatMessage[];
}

const mutationActions = new Set<string>([
  "resource_created", "resource_updated", "resource_deleted",
  "category_created", "category_updated", "category_deleted",
  "todo_created", "todo_updated", "todo_deleted",
  "reminder_created", "reminder_updated", "reminder_deleted",
]);

function uid(prefix: string): string {
  return `${prefix}${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

function seedConversations(): Conversation[] {
  return DEMO_CONVERSATIONS.map((c) => ({
    id: c.id,
    title: c.title,
    archived: false,
    messages: c.messages.map((m) => ({
      id: m.id,
      role: m.role,
      content: m.content,
      createdAt: "",
    })),
  }));
}

interface ChatState {
  conversations: Conversation[];
  isSending: boolean;
  error: string | null;

  newConversation: () => string;
  renameConversation: (id: string, title: string) => void;
  archiveConversation: (id: string) => void;
  deleteConversation: (id: string) => string | null; // returns fallback dockConvId
  sendToConversation: (convId: string, text: string) => Promise<void>;
}

export const useChatStore = create<ChatState>((set, get) => ({
  conversations: seedConversations(),
  isSending: false,
  error: null,

  newConversation: () => {
    const id = uid("cv");
    set((s) => ({
      conversations: [{ id, title: "New conversation", archived: false, messages: [] }, ...s.conversations],
    }));
    return id;
  },

  renameConversation: (id, title) => {
    const v = title.trim();
    if (!v) return;
    set((s) => ({ conversations: s.conversations.map((c) => (c.id === id ? { ...c, title: v } : c)) }));
  },

  archiveConversation: (id) => {
    set((s) => ({ conversations: s.conversations.map((c) => (c.id === id ? { ...c, archived: true } : c)) }));
  },

  deleteConversation: (id) => {
    const rest = get().conversations.filter((c) => c.id !== id);
    set({ conversations: rest });
    return (rest.find((c) => !c.archived) ?? rest[0])?.id ?? null;
  },

  sendToConversation: async (convId, text) => {
    const v = text.trim();
    if (v === "") return;

    const userMsg: ChatMessage = { id: uid("u"), role: "user", content: v, createdAt: new Date().toISOString() };
    set((s) => ({
      conversations: s.conversations.map((c) => (c.id === convId ? { ...c, messages: [...c.messages, userMsg] } : c)),
      isSending: true,
      error: null,
    }));

    try {
      const result = await sendChatCommand(v);
      const reply = result.message?.trim() || `Searching your graph for “${v}” — ranked by counter weighting.`;
      const aiMsg: ChatMessage = { id: uid("a"), role: "assistant", content: reply, createdAt: new Date().toISOString() };
      set((s) => ({
        conversations: s.conversations.map((c) => (c.id === convId ? { ...c, messages: [...c.messages, aiMsg] } : c)),
        isSending: false,
      }));
      if (mutationActions.has(result.action)) {
        void useResourceStore.getState().loadResources({ silent: true });
        void useTaskStore.getState().loadAll({ silent: true });
      }
    } catch {
      // offline / no backend → canned reply so the chat still works in demo mode
      const aiMsg: ChatMessage = {
        id: uid("a"),
        role: "assistant",
        content: `Searching your graph for “${v}” — 3 resources matched, ranked by counter weighting.`,
        createdAt: new Date().toISOString(),
      };
      set((s) => ({
        conversations: s.conversations.map((c) => (c.id === convId ? { ...c, messages: [...c.messages, aiMsg] } : c)),
        isSending: false,
      }));
    }
  },
}));
