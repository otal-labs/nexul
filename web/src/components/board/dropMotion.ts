import type { DropAnimation } from "@dnd-kit/core";

import { EASE_OUT, prefersReducedMotion } from "@/lib/motion";

interface Landing {
  // The card left a working column for a done one: it washes in the success hue once it lands.
  done: boolean;
  // A keyboard drag keeps the glide home (it says where the card went); the wash is news, so it stays too.
  keyboard: boolean;
}

// One board drags at a time; the drag's start and end write here and the drop animation reads it once the card lands.
const landing: Landing = { done: false, keyboard: false };
export const startLanding = (keyboard: boolean) => Object.assign(landing, { done: false, keyboard });
export const landsInDone = (done: boolean) => (landing.done = done);

const wash = (card: HTMLElement) => {
  const layer = card.querySelector<HTMLElement>("[data-wash]");
  if (!layer || typeof layer.animate !== "function") return;
  layer.animate([{ opacity: 1 }, { opacity: 0 }], { duration: prefersReducedMotion() ? 400 : 800, easing: "ease-out" });
};

// Settle (the Board drop lock in practices/design-language.md): the lifted card comes down as it glides onto its slot.
export const dropAnimationFor = (): DropAnimation => ({
  duration: 220,
  easing: EASE_OUT,
  sideEffects: ({ active, dragOverlay }) => {
    const card = dragOverlay.node.firstElementChild as HTMLElement | null;
    if (card) card.style.scale = "1";
    const shadow = card?.querySelector<HTMLElement>("[data-lift-shadow]");
    if (shadow) shadow.style.opacity = "0";
    active.node.style.opacity = "0";
    return () => {
      active.node.style.opacity = "";
      if (landing.done) wash(active.node);
    };
  },
});
