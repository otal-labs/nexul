import { create } from "zustand";
import { persist } from "zustand/middleware";

export type ChatDraftStore = {
  // Unsent composer text per conversation id, so leaving a chat keeps what was typed there.
  drafts: Record<string, string>;
  setDraft: (conversationId: string, text: string) => void;
};

export const useChatDraftStore = create<ChatDraftStore>()(
  persist(
    (set) => ({
      drafts: {},
      setDraft: (conversationId, text) =>
        set((s) => {
          const rest = Object.fromEntries(Object.entries(s.drafts).filter(([id]) => id !== conversationId));
          if (text === "") return { drafts: rest };
          return { drafts: { ...rest, [conversationId]: text } };
        }),
    }),
    { name: "chat-drafts", partialize: (s) => ({ drafts: s.drafts }) },
  ),
);
