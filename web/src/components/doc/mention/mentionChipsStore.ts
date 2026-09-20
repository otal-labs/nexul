import type { MentionChipData } from "@/models/Mention";

type Listener = () => void;

// Cross-root store: NodeViews in separate ProseMirror roots subscribe for live title/status.
let chips = new Map<string, MentionChipData>();
const listeners = new Set<Listener>();

export function publishChips(next: Map<string, MentionChipData>): void {
  chips = next;
  for (const listener of listeners) listener();
}

export function subscribeChips(listener: Listener): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

export function getChips(): Map<string, MentionChipData> {
  return chips;
}
