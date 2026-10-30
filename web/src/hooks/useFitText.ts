import { useLayoutEffect, useRef, useState } from "react";

import type { Measure } from "@/utils/FitTextUtility";

let canvas: CanvasRenderingContext2D | null | undefined;

const px = (value: string) => Number.parseFloat(value) || 0;

// The parent's content width less every visible sibling and the gaps between them: what the text may take.
const spaceFor = (element: HTMLElement): number => {
  const parent = element.parentElement;
  if (!parent) return 0;
  const style = getComputedStyle(parent);
  const siblings = (Array.from(parent.children) as HTMLElement[]).filter(
    (child) => child !== element && getComputedStyle(child).display !== "none",
  );
  const taken = siblings.reduce((sum, child) => {
    const cs = getComputedStyle(child);
    return sum + child.getBoundingClientRect().width + px(cs.marginLeft) + px(cs.marginRight);
  }, 0);
  return parent.clientWidth - px(style.paddingLeft) - px(style.paddingRight) - taken - px(style.columnGap) * siblings.length;
};

// The cut of `full` that fits beside its siblings, measured on a canvas; `fit` must be module-level, or every render re-measures.
export const useFitText = <T extends HTMLElement>(full: string, fit: (text: string, width: number, measure: Measure) => string) => {
  const ref = useRef<T>(null);
  const [shown, setShown] = useState(full);
  useLayoutEffect(() => {
    const element = ref.current;
    const parent = element?.parentElement;
    if (!element || !parent) return;
    const update = () => {
      const width = spaceFor(element) - 1;
      if (width <= 0) return setShown(full);
      canvas ??= document.createElement("canvas").getContext("2d");
      if (!canvas) return setShown(full);
      const ctx = canvas;
      const cs = getComputedStyle(element);
      ctx.font = `${cs.fontStyle} ${cs.fontWeight} ${cs.fontSize} ${cs.fontFamily}`;
      setShown(fit(full, width, (text) => ctx.measureText(text).width));
    };
    update();
    const observer = new ResizeObserver(update);
    observer.observe(parent);
    for (const child of Array.from(parent.children)) if (child !== element) observer.observe(child);
    void document.fonts?.ready.then(update);
    return () => observer.disconnect();
  }, [full, fit]);
  return [ref, shown] as const;
};
