import { EASE_OUT, prefersReducedMotion } from "@/lib/motion";

// The scroller re-attaches the item's ref when the server copy confirms it; the send plays once per row.
const played = new WeakSet<HTMLElement>();

// Your sent message rises 12px into place while its bubble grows from 0.96 out of its bottom-right corner, 240ms.
export const playSend = (item: HTMLElement | null) => {
  const bubble = item?.querySelector<HTMLElement>('[data-slot="bubble"]');
  if (!item || !bubble || played.has(item) || typeof item.animate !== "function") return;
  played.add(item);
  if (prefersReducedMotion()) {
    item.animate([{ opacity: 0 }, { opacity: 1 }], { duration: 150, easing: EASE_OUT });
    return;
  }
  item.animate([{ opacity: 0, translate: "0 12px" }, { opacity: 1, translate: "0 0" }], { duration: 240, easing: EASE_OUT });
  bubble.style.transformOrigin = "bottom right";
  bubble.animate([{ scale: 0.96 }, { scale: 1 }], { duration: 240, easing: EASE_OUT });
};
