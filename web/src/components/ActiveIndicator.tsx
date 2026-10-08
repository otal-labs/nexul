import { useLayoutEffect, useRef, type ReactNode } from "react";

import { lastInputWasKeyboard, prefersReducedMotion } from "@/lib/motion";
import { cn } from "@/lib/utils";

interface ActiveIndicatorProps {
  /** Finds the active item inside the indicator's parent, e.g. `[aria-current="page"]`. */
  selector: string;
  className?: string;
  children?: ReactNode;
}

// One highlight per list that slides to the active item, played back from where it is so a change mid-slide carries on; parent needs `relative isolate`.
export const ActiveIndicator = ({ selector, className, children }: ActiveIndicatorProps) => {
  const ref = useRef<HTMLSpanElement>(null);

  useLayoutEffect(() => {
    const el = ref.current;
    const parent = el?.parentElement;
    if (!el || !parent) return;
    let current: Element | null = null;
    let last = "";

    const place = () => {
      const target = parent.querySelector(selector);
      if (!target) {
        if (current) el.style.opacity = "0";
        current = null;
        return;
      }
      const box = parent.getBoundingClientRect();
      const to = target.getBoundingClientRect();
      const x = to.left - box.left - parent.clientLeft + parent.scrollLeft;
      const y = to.top - box.top - parent.clientTop + parent.scrollTop;
      const key = `${x},${y},${to.width},${to.height}`;
      if (target === current && key === last) return;
      const from = el.getBoundingClientRect();
      // A key press (arrowing through tabs) moves the highlight at once, never trailing the keys.
      const slide = current !== null && current !== target && !prefersReducedMotion() && !lastInputWasKeyboard();
      const appear = current === null;
      current = target;
      last = key;

      const rest = `translate(${x}px, ${y}px)`;
      el.style.transition = "none";
      el.style.width = `${to.width}px`;
      el.style.height = `${to.height}px`;
      el.style.transform = slide
        ? `translate(${x + from.left - to.left}px, ${y + from.top - to.top}px) scale(${from.width / to.width}, ${from.height / to.height})`
        : rest;
      if (appear) el.style.opacity = "0";
      // Commit the start frame so the class transition runs from it.
      void el.offsetWidth;
      el.style.transition = "";
      el.style.transform = rest;
      el.style.opacity = "1";
    };

    place();
    const resize = new ResizeObserver(place);
    resize.observe(parent);
    const mutations = new MutationObserver(place);
    mutations.observe(parent, { subtree: true, childList: true, attributes: true, attributeFilter: ["aria-current", "data-state"] });
    return () => {
      resize.disconnect();
      mutations.disconnect();
    };
  }, [selector]);

  return (
    <span
      ref={ref}
      aria-hidden
      className={cn(
        "pointer-events-none absolute top-0 left-0 -z-10 origin-top-left opacity-0 transition-[transform,opacity] duration-200 ease-spring",
        className,
      )}
    >
      {children}
    </span>
  );
};

// The line tab row's underline, sliding to the active tab; the row's triggers hide their own.
export const TabUnderline = () => (
  <ActiveIndicator selector='[role="tab"][data-state="active"]'>
    <span className="absolute inset-x-0 -bottom-px h-0.5 bg-foreground" />
  </ActiveIndicator>
);
