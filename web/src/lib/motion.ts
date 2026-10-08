// The JavaScript half of the Motion baseline in practices/design-language.md; the CSS half is in index.css.
export const EASE_OUT = "cubic-bezier(0.16, 1, 0.3, 1)";

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

// A block that holds an entering list leaves the motion to its rows; one that runs its own entrance keeps it.
const pageBlocks = (frame: HTMLElement): Element[] => {
  const panels = isPanel(frame) ? [frame.firstElementChild].filter((el) => el !== null) : findPanels(frame, 4);
  return panels
    .flatMap((panel) => [...panel.children].flatMap(unwrap))
    .filter((block) => !block.matches("[data-enter-list]") && !block.querySelector("[data-enter-list]") && (block.getAnimations?.().length ?? 0) === 0);
};

// The page's content rises inside its panels, which never move; blocks that mount while its data lands rise as they arrive.
export const enterPage = (frame: HTMLElement) => {
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
