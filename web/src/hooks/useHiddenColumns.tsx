import { useEffect, useRef, useState } from "react";

// Columns starting past the scroller's edge less its fade, kept current as it scrolls and resizes.
export const useHiddenColumns = (fade: number) => {
  const ref = useRef<HTMLDivElement>(null);
  const [hidden, setHidden] = useState(0);
  useEffect(() => {
    const scroller = ref.current;
    if (!scroller) return;
    const count = () => {
      const edge = scroller.getBoundingClientRect().right - fade;
      setHidden([...scroller.querySelectorAll("[data-role-head]")].filter((head) => head.getBoundingClientRect().left >= edge - 1).length);
    };
    count();
    const observer = typeof ResizeObserver === "undefined" ? undefined : new ResizeObserver(count);
    observer?.observe(scroller);
    scroller.addEventListener("scroll", count, { passive: true });
    return () => {
      observer?.disconnect();
      scroller.removeEventListener("scroll", count);
    };
  }, [fade]);
  const more = () => {
    const still = window.matchMedia?.("(prefers-reduced-motion: reduce)").matches;
    ref.current?.scrollBy({ left: ref.current.clientWidth * 0.75, behavior: still ? "auto" : "smooth" });
  };
  return { ref, hidden, more };
};
