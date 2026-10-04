import { useEffect } from "react";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { useSearchParams } from "react-router";

import { STATES, VARIANTS } from "@/components/memory/prototype/SourcesProtoStore";

interface SourcesProtoSwitcherProps {
  variant: number;
  state: number;
}

const IGNORE = "input, textarea, select, [contenteditable=''], [contenteditable='true'], [role=radio], [role=tab], [role=slider]";

const Stepper = ({ label, onStep, title }: { label: string; onStep: (d: number) => void; title: string }) => (
  <span className="flex items-center" title={title}>
    <button type="button" aria-label={`Previous ${title}`} onClick={() => onStep(-1)} className="rounded p-1 hover:bg-background/15">
      <ChevronLeft className="size-3.5" />
    </button>
    <span className="min-w-0 px-1 whitespace-nowrap">{label}</span>
    <button type="button" aria-label={`Next ${title}`} onClick={() => onStep(1)} className="rounded p-1 hover:bg-background/15">
      <ChevronRight className="size-3.5" />
    </button>
  </span>
);

// Dev-only prototype switcher: ←/→ cycle the variant, ↑/↓ the state, both kept in the URL.
export const SourcesProtoSwitcher = ({ variant, state }: SourcesProtoSwitcherProps) => {
  const [, setParams] = useSearchParams();

  useEffect(() => {
    const go = (v: number, s: number) =>
      setParams(
        (p) => {
          p.set("variant", VARIANTS[v]!.key);
          p.set("state", String(s));
          return p;
        },
        { replace: true },
      );
    const step = { ArrowLeft: [-1, 0], ArrowRight: [1, 0], ArrowUp: [0, -1], ArrowDown: [0, 1] } as const;
    const onKey = (e: KeyboardEvent) => {
      const d = step[e.key as keyof typeof step];
      if (!d || e.defaultPrevented || e.metaKey || e.ctrlKey || e.altKey) return;
      if (e.target instanceof Element && e.target.closest(IGNORE)) return;
      e.preventDefault();
      go((variant + d[0] + VARIANTS.length) % VARIANTS.length, ((state - 1 + d[1] + STATES.length) % STATES.length) + 1);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [variant, state, setParams]);

  const press = (key: string) => () => window.dispatchEvent(new KeyboardEvent("keydown", { key }));

  return (
    <div className="sticky bottom-4 z-50 mx-auto mt-12 flex w-fit items-center gap-2 rounded-lg bg-foreground px-2 py-1.5 font-mono text-xs text-background shadow-overlay">
      <Stepper
        title="variant"
        label={`${VARIANTS[variant]!.key} · ${VARIANTS[variant]!.label}`}
        onStep={(d) => press(d < 0 ? "ArrowLeft" : "ArrowRight")()}
      />
      <span className="h-4 w-px bg-background/30" aria-hidden />
      <Stepper title="state" label={`${state} · ${STATES[state - 1]}`} onStep={(d) => press(d < 0 ? "ArrowUp" : "ArrowDown")()} />
    </div>
  );
};
