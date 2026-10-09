import { EASE_OUT, EASE_STANDARD, prefersReducedMotion } from "@/lib/motion";

interface Box {
  left: number;
  width: number;
  count: string;
}

const last = new WeakMap<HTMLElement, Map<string, Box>>();

const measure = (summary: HTMLElement, segments: HTMLElement[]) => {
  const bar = summary.querySelector("[data-stage-bar]")?.getBoundingClientRect();
  const boxes = new Map<string, Box>();
  for (const seg of segments) {
    const key = seg.dataset.stageSegment ?? "";
    const r = seg.getBoundingClientRect();
    boxes.set(key, { left: r.left - (bar?.left ?? 0), width: r.width, count: countOf(summary, key)?.textContent ?? "" });
  }
  return boxes;
};

const countOf = (summary: HTMLElement, key: string) => summary.querySelector<HTMLElement>(`[data-stage-count="${key}"]`);

const grow = (seg: HTMLElement, duration: number, easing: string) =>
  seg.animate([{ transform: "scaleX(0)" }, { transform: "none" }], { duration, easing });

// Segments grow and glide one by one, never the bar as a whole (the Board stage bar in design-language.md's Motion baseline).
export const playStageBar = (summary: HTMLElement | null) => {
  if (!summary || typeof summary.animate !== "function") return;
  const segments = [...summary.querySelectorAll<HTMLElement>("[data-stage-segment]")];
  const before = last.get(summary);
  const after = measure(summary, segments);
  last.set(summary, after);
  const reduced = prefersReducedMotion();
  if (!before) {
    if (!reduced) segments.forEach((seg) => grow(seg, 300, EASE_OUT));
    return;
  }
  for (const seg of segments) {
    const key = seg.dataset.stageSegment ?? "";
    const from = before.get(key);
    const to = after.get(key);
    if (!to) continue;
    if (from && from.count !== to.count)
      countOf(summary, key)?.animate([{ opacity: 0, transform: reduced ? "none" : "translateY(6px)" }, { opacity: 1, transform: "none" }], {
        duration: reduced ? 150 : 200,
        easing: EASE_OUT,
      });
    if (reduced) continue;
    if (!from) {
      grow(seg, 250, EASE_STANDARD);
      continue;
    }
    if (from.left === to.left && from.width === to.width) continue;
    seg.animate([{ transform: `translateX(${from.left - to.left}px) scaleX(${from.width / to.width})` }, { transform: "none" }], {
      duration: 250,
      easing: EASE_STANDARD,
    });
  }
};
