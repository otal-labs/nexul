// The JavaScript half of the Motion baseline in practices/design-language.md; the CSS half is in index.css.
export const EASE_OUT = "cubic-bezier(0.16, 1, 0.3, 1)";
export const EASE_STANDARD = "cubic-bezier(0.25, 0.1, 0.25, 1)";
export const SPRING_POP = "linear(0, 0.1, 0.303, 0.515, 0.693, 0.826, 0.915, 0.969, 0.998, 1.011, 1.015, 1.014, 1.011, 1.008, 1.005, 1.003, 1.002, 1.001, 1)";

const STAGGER_STEP_MS = 25;
const STAGGER_ROWS = 8;
// A list this long mounts at once: a cascade over fifty rows is lag, not motion.
const STAGGER_CEILING = 50;

const PAGE_STAGGER_MS = 30;
const PAGE_BLOCKS = 3;
const PAGE_WINDOW_MS = 400;

export const prefersReducedMotion = () => window.matchMedia?.("(prefers-reduced-motion: reduce)").matches ?? false;

// Whether the last thing that could have moved the page or a selection was a key rather than a pointer.
const MOVE_KEYS = new Set(["Enter", " ", "ArrowLeft", "ArrowRight", "ArrowUp", "ArrowDown", "Home", "End", "PageUp", "PageDown"]);
let keyboardLast = false;
window.addEventListener("keydown", (event) => (keyboardLast = MOVE_KEYS.has(event.key)), true);
window.addEventListener("pointerdown", () => (keyboardLast = false), true);
export const lastInputWasKeyboard = () => keyboardLast;

// Rows past the eighth arrive with it, so the tail never shows before the cascade reaches it.
const staggerDelayMs = (index: number) => Math.min(index, STAGGER_ROWS) * STAGGER_STEP_MS;

const fade: Keyframe[] = [{ opacity: 0 }, { opacity: 1 }];
const rise = (px: number): Keyframe[] => (prefersReducedMotion() ? fade : [{ opacity: 0, translate: `0 ${px}px` }, { opacity: 1, translate: "0 0" }]);
const pop = (): Keyframe[] => (prefersReducedMotion() ? fade : [{ opacity: 0, scale: 0.97 }, { opacity: 1, scale: 1 }]);

// Reduced motion keeps the fade that says something arrived and drops the travel and the cascade.
const enter = (el: Element, keyframes: Keyframe[], duration: number, delay = 0) => {
  if (typeof el.animate !== "function") return;
  const reduced = prefersReducedMotion();
  el.animate(keyframes, { duration: reduced ? 150 : duration, delay: reduced ? 0 : delay, easing: EASE_OUT, fill: "backwards" });
};

// A section opened by a click settles in like a disclosure's content: 4px over 200ms.
export const settleIn = (el: Element | null | undefined) => el && enter(el, rise(4), 200);

const isRow = (node: Node): node is HTMLElement => node instanceof HTMLElement && !node.hasAttribute("data-no-enter");

export type Arrival = "pop" | "rise";

// Rows at mount cascade in, rows added later pop or rise in; a row React only moved shows as removed and added, and keeps still.
export const enterList = (list: HTMLElement, arrival: Arrival = "pop") => {
  const rows = [...list.children].filter(isRow);
  if (rows.length < STAGGER_CEILING) rows.forEach((row, i) => enter(row, rise(4), 200, staggerDelayMs(i)));
  const observer = new MutationObserver((records) => {
    const moved = new Set(records.flatMap((r) => [...r.removedNodes]));
    for (const node of records.flatMap((r) => [...r.addedNodes])) {
      if (!isRow(node) || moved.has(node) || node.parentElement !== list) continue;
      if (arrival === "rise") enter(node, rise(8), 200);
      if (arrival === "pop") enter(node, pop(), 150);
    }
  });
  observer.observe(list, { childList: true });
  return () => observer.disconnect();
};

const isPanel = (el: Element) => {
  const style = getComputedStyle(el);
  return style.display !== "contents" && style.backdropFilter !== "none" && style.backdropFilter !== "";
};

// The frosted panels a page is built from; the frame itself is the panel of a single-surface page.
const findPanels = (el: Element, depth: number): Element[] =>
  [...el.children].flatMap((child) => {
    if (isPanel(child)) return [child];
    return depth > 0 ? findPanels(child, depth - 1) : [];
  });

const unwrap = (el: Element): Element[] => (getComputedStyle(el).display === "contents" ? [...el.children].flatMap(unwrap) : [el]);

// A block that holds an entering list or an empty state leaves the motion to them; one that runs its own entrance keeps it.
const OWN_ENTRANCE = "[data-enter-list], [data-enter-own]";
const pageBlocks = (frame: HTMLElement): Element[] => {
  const panels = isPanel(frame) ? [frame.firstElementChild].filter((el) => el !== null) : findPanels(frame, 4);
  return panels
    .flatMap((panel) => [...panel.children].flatMap(unwrap))
    .filter((block) => !block.matches(OWN_ENTRANCE) && !block.querySelector(OWN_ENTRANCE) && (block.getAnimations?.().length ?? 0) === 0);
};

let pageEnteredAt = -Infinity;

// Content replacing a loader that was on screen rises in like a late page block; never a panel, since panels never move.
export const enterAfterLoader = (parent: HTMLElement, before: Set<Element>) => {
  if (performance.now() - pageEnteredAt < PAGE_WINDOW_MS || !parent.closest(".app-frame")) return;
  [...parent.children]
    .filter((child) => !before.has(child) && !isPanel(child) && !child.matches(OWN_ENTRANCE) && !child.querySelector(OWN_ENTRANCE))
    .forEach((child) => enter(child, rise(6), 200));
};

// The page's content rises inside its panels, which never move; blocks that mount while its data lands rise as they arrive.
export const enterPage = (frame: HTMLElement) => {
  pageEnteredAt = performance.now();
  const seen = new WeakSet<Element>();
  const run = (stagger: boolean) =>
    pageBlocks(frame)
      .filter((block) => !seen.has(block))
      .forEach((block, i) => {
        seen.add(block);
        enter(block, rise(6), 200, stagger ? Math.min(i, PAGE_BLOCKS - 1) * PAGE_STAGGER_MS : 0);
      });
  run(true);
  const observer = new MutationObserver(() => run(false));
  observer.observe(frame, { childList: true, subtree: true });
  const timer = window.setTimeout(() => observer.disconnect(), PAGE_WINDOW_MS);
  return () => {
    window.clearTimeout(timer);
    observer.disconnect();
  };
};

export type SwapAxis = "x" | "y";

const SWAP_DISTANCE_PX = 6;
const SWAP_MS = 160;

// A tab or section's new content arrives from the side of the item it was picked from; nothing leaves, so input never waits.
export const swapIn = (el: Element | null | undefined, step: number, axis: SwapAxis) => {
  if (!el || typeof el.animate !== "function" || lastInputWasKeyboard()) return;
  if (prefersReducedMotion()) {
    el.animate(fade, { duration: 120, easing: EASE_OUT });
    return;
  }
  // One entrance per container: a list that cascaded in with the new content arrives with it instead.
  for (const running of el.getAnimations?.({ subtree: true }) ?? []) {
    if (running.effect?.getTiming().iterations !== Infinity) running.finish();
  }
  const offset = Math.sign(step) * SWAP_DISTANCE_PX;
  const from = axis === "x" ? `${offset}px 0` : `0 ${offset}px`;
  el.animate([{ opacity: 0, translate: from }, { opacity: 1, translate: "0 0" }], { duration: SWAP_MS, easing: EASE_OUT });
};

const RAIL_PX = 68;
const REVEAL_MS = 220;

// The open sidebar uncovers itself left to right from the rail's edge; clip-path is composited, so the panel beside it
// takes its new width in one layout step and nothing reflows while it plays. Closing snaps.
export const revealSidebar = (el: HTMLElement | null, open: boolean) => {
  if (!el || typeof el.animate !== "function") return;
  // Closed mid-reveal, the rail must not stay clipped to nothing.
  el.getAnimations().forEach((running) => running.cancel());
  if (!open || prefersReducedMotion() || lastInputWasKeyboard()) return;
  const hidden = Math.max(0, el.offsetWidth - RAIL_PX);
  el.animate([{ clipPath: `inset(0 ${hidden}px 0 0)` }, { clipPath: "inset(0 0 0 0)" }], { duration: REVEAL_MS, easing: EASE_OUT });
};
