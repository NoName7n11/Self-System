import { create } from "zustand";

import { sendChatCommand } from "../api/client";
import { demoChatMessages } from "../lib/demoData";
import type { ChatMessage } from "../types";
import { useResourceStore } from "./useResourceStore";
import { useTaskStore } from "./useTaskStore";

const mutationActions = new Set<string>([
  "resource_created",
  "resource_updated",
  "resource_deleted",
  "category_created",
  "category_updated",
  "category_deleted",
  "todo_created",
  "todo_updated",
  "todo_deleted",
  "reminder_created",
  "reminder_updated",
  "reminder_deleted",
]);

function buildMessage(role: ChatMessage["role"], content: string): ChatMessage {
  return {
    id: `${Date.now()}-${Math.random().toString(16).slice(2)}`,
    role,
    content,
    createdAt: new Date().toISOString(),
  };
}

interface ChatState {
  messages: ChatMessage[];
  isSending: boolean;
  error: string | null;
  sendMessage: (content: string) => Promise<void>;
}

export const useChatStore = create<ChatState>((set) => ({
  // seed with the demo conversation so the dock Chat tab looks populated
  messages: demoChatMessages(),
  isSending: false,
  error: null,

  sendMessage: async (content) => {
    const trimmed = content.trim();
    if (trimmed === "") {
      return;
    }

    const userMessage = buildMessage("user", trimmed);
    set((state) => ({
      messages: [...state.messages, userMessage],
      isSending: true,
      error: null,
    }));

    try {
      const result = await sendChatCommand(trimmed);
      const assistantText = result.message?.trim() || `Command executed (${result.action || "unknown"}).`;

      set((state) => ({
        messages: [...state.messages, buildMessage("assistant", assistantText)],
        isSending: false,
        error: null,
      }));

      if (mutationActions.has(result.action)) {
        void useResourceStore.getState().loadResources({ silent: true });
        void useTaskStore.getState().loadAll({ silent: true });
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : "Failed to run chat command";
      set((state) => ({
        messages: [...state.messages, buildMessage("assistant", `Command failed: ${message}`)],
        isSending: false,
        error: message,
      }));
    }
  },
}));
