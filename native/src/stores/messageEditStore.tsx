import { create } from "zustand";

export interface EditingMessage {
  conversationId: string;
  messageId: string;
  body: string;
}

export type MessageEditStore = {
  editing: EditingMessage | null;
  start: (editing: EditingMessage) => void;
  stop: () => void;
};

// Which of your messages the composer is rewriting; not persisted, so a relaunch opens on a plain composer.
export const useMessageEditStore = create<MessageEditStore>((set) => ({
  editing: null,
  start: (editing) => set({ editing }),
  stop: () => set({ editing: null }),
}));
