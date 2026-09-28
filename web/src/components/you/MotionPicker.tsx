import { useEffect } from "react";
import { useShallow } from "zustand/react/shallow";

import { cn } from "@/lib/utils";
import { type ConnectVariant, usePrototypeStore } from "@/components/you/prototypeStore";

const VARIANTS: { value: ConnectVariant; label: string }[] = [
  { value: "swap", label: "1 · Card confirms" },
  { value: "flash", label: "2 · Card + row glow" },
  { value: "toast", label: "3 · Toast only" },
];

const SPEEDS = [1, 0.5, 0.25];

const chip = (active: boolean) =>
  cn("rounded-md border px-2 py-1 text-xs", active ? "border-ring bg-accent text-foreground" : "text-muted-foreground hover:bg-accent/50");

export const MotionPicker = () => {
  const { variant, speed, setVariant, setSpeed, scan, reset } = usePrototypeStore(
    useShallow((s) => ({ variant: s.variant, speed: s.speed, setVariant: s.setVariant, setSpeed: s.setSpeed, scan: s.scan, reset: s.reset })),
  );

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.target instanceof HTMLInputElement) return;
      if (e.key === "r" || e.key === "R") {
        reset();
        setTimeout(scan, 400);
      }
      if (e.key === "1") setVariant("swap");
      if (e.key === "2") setVariant("flash");
      if (e.key === "3") setVariant("toast");
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [reset, scan, setVariant]);

  useEffect(() => {
    const timer = setInterval(() => {
      document.getAnimations().forEach((a) => {
        a.playbackRate = speed;
      });
    }, 16);
    return () => clearInterval(timer);
  }, [speed]);

  return (
    <div className="fixed right-4 bottom-4 z-50 w-72 space-y-3 rounded-lg border bg-popover p-3 shadow-overlay">
      <p className="font-mono text-[10px] tracking-[0.14em] text-muted-foreground uppercase">Phone connected · pick one</p>
      <div className="flex flex-col gap-1">
        {VARIANTS.map((v) => (
          <button key={v.value} type="button" className={chip(variant === v.value)} onClick={() => setVariant(v.value)}>
            {v.label}
          </button>
        ))}
      </div>
      <div className="flex gap-1">
        {SPEEDS.map((s) => (
          <button key={s} type="button" className={chip(speed === s)} onClick={() => setSpeed(s)}>
            {s}x
          </button>
        ))}
      </div>
      <div className="flex gap-1">
        <button type="button" className={chip(false)} onClick={scan}>
          Simulate scan
        </button>
        <button type="button" className={chip(false)} onClick={reset}>
          Reset
        </button>
      </div>
      <p className="text-[11px] text-muted-foreground">R replays · 1 2 3 switch</p>
    </div>
  );
};
