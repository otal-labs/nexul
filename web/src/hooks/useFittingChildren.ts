import { useLayoutEffect, useRef, useState } from "react";

interface Fit {
  fitting: number;
  slack: number;
  end: number;
}

// Children on a wrapping container's first row, the free width after them, and where the last one ends.
export const useFittingChildren = <T extends HTMLElement>(count: number) => {
  const ref = useRef<T>(null);
  const [fit, setFit] = useState<Fit>({ fitting: count, slack: 0, end: 0 });
  useLayoutEffect(() => {
    const element = ref.current;
    if (!element) return;
    const measure = () => {
      const children = Array.from(element.children) as HTMLElement[];
      const first = children.findIndex((child) => child.offsetTop > 0 || child.offsetLeft + child.offsetWidth > element.clientWidth);
      const fitting = first === -1 ? children.length : first;
      const last = children[fitting - 1];
      const end = last ? last.offsetLeft + last.offsetWidth : 0;
      const slack = element.clientWidth - end;
      setFit((previous) => (previous.fitting === fitting && previous.slack === slack && previous.end === end ? previous : { fitting, slack, end }));
    };
    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(element);
    return () => observer.disconnect();
  }, [count]);
  return [ref, Math.min(fit.fitting, count), fit.slack, fit.end] as const;
};
