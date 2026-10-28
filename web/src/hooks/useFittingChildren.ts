import { useLayoutEffect, useRef, useState } from "react";

// How many of a one-row wrapping container's children sit whole on its first row; the rest wrapped or overflow.
export const useFittingChildren = <T extends HTMLElement>(count: number) => {
  const ref = useRef<T>(null);
  const [fitting, setFitting] = useState(count);
  useLayoutEffect(() => {
    const element = ref.current;
    if (!element) return;
    const measure = () => {
      const children = Array.from(element.children) as HTMLElement[];
      const first = children.findIndex((child) => child.offsetTop > 0 || child.offsetLeft + child.offsetWidth > element.clientWidth);
      setFitting(first === -1 ? children.length : first);
    };
    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(element);
    return () => observer.disconnect();
  }, [count]);
  return [ref, Math.min(fitting, count)] as const;
};
