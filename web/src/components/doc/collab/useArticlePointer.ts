import { useRef, type PointerEvent } from "react";

import type { CollabSession } from "@/components/doc/collab/useCollabSession";

// Shares this person's pointer over the article with the session's peers, about 20 times a second.
export const useArticlePointer = (session: CollabSession | null | undefined) => {
  const articleRef = useRef<HTMLElement | null>(null);
  const lastSent = useRef(0);

  const onPointerMove = (e: PointerEvent<HTMLElement>) => {
    if (!session) return;
    const now = performance.now();
    if (now - lastSent.current < 50) return;
    lastSent.current = now;
    const rect = articleRef.current?.getBoundingClientRect();
    if (!rect) return;
    session.provider.awareness.setLocalStateField("pointer", {
      x: Math.round(e.clientX - rect.left),
      y: Math.round(e.clientY - rect.top),
    });
  };

  const onPointerLeave = () => session?.provider.awareness.setLocalStateField("pointer", null);

  return { articleRef, onPointerMove, onPointerLeave };
};
