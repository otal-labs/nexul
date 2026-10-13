import { useEffect, useLayoutEffect, useRef, useState } from "react";

import "./picker.css";

interface PickerProps {
  names: string[];
  active: number;
  onPick: (i: number) => void;
  onReplay: () => void;
}

export const Picker = ({ names, active, onPick, onReplay }: PickerProps) => {
  const nav = useRef<HTMLElement>(null);
  const items = useRef<(HTMLButtonElement | null)[]>([]);
  const highlight = useRef<HTMLSpanElement>(null);
  const [ready, setReady] = useState(false);

  useLayoutEffect(() => {
    const move = () => {
      const el = items.current[active];
      if (!el || !highlight.current) return;
      highlight.current.style.width = `${el.offsetWidth}px`;
      highlight.current.style.transform = `translateX(${el.offsetLeft}px)`;
    };
    move();
    window.addEventListener("resize", move);
    return () => window.removeEventListener("resize", move);
  }, [active]);

  useEffect(() => {
    requestAnimationFrame(() => requestAnimationFrame(() => setReady(true)));
  }, []);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const t = e.target as HTMLElement;
      if (/^(INPUT|TEXTAREA|SELECT)$/.test(t.tagName) || t.isContentEditable) return;
      if (e.metaKey || e.ctrlKey || e.altKey) return;
      const num = parseInt(e.key, 10);
      if (num >= 1 && num <= names.length) onPick(num - 1);
      if (e.key === "ArrowRight") onPick((active + 1) % names.length);
      if (e.key === "ArrowLeft") onPick((active - 1 + names.length) % names.length);
      if (e.key === "r" || e.key === "R") onReplay();
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [active, names.length, onPick, onReplay]);

  return (
    <nav ref={nav} className="proto-picker" aria-label="Prototype variants" data-ready={ready ? "" : undefined}>
      <span ref={highlight} className="proto-picker-highlight" aria-hidden="true" />
      {names.map((name, i) => (
        <button
          key={name}
          ref={(el) => {
            items.current[i] = el;
          }}
          className="proto-picker-item"
          data-active={i === active ? "" : undefined}
          aria-current={i === active ? "true" : undefined}
          onClick={() => onPick(i)}
        >
          {name}
        </button>
      ))}
      <span className="proto-picker-divider" aria-hidden="true" />
      <button className="proto-picker-item proto-picker-replay" aria-label="Replay animation (R)" onClick={onReplay}>
        ↻
      </button>
    </nav>
  );
};
