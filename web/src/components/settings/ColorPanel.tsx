import { useState, type KeyboardEvent, type PointerEvent } from "react";

import { NAMED_HEX } from "@/components/board/ticketTypeColor";
import { cn } from "@/lib/utils";
import { HEX_COLOR, hexToHsv, hsvToHex, type Hsv } from "@/lib/hsv";

const QUICK_HUES = [
  "#ef4444", "#f97316", "#f59e0b", "#84cc16", "#22c55e", "#10b981", "#14b8a6", "#06b6d4",
  "#0ea5e9", "#3b82f6", "#6366f1", "#8b5cf6", "#a855f7", "#d946ef", "#ec4899", "#f43f5e", "#64748b",
];

const HUE_STRIP = "linear-gradient(to right, #f00, #ff0, #0f0, #0ff, #00f, #f0f, #f00)";
const clamp = (n: number) => Math.min(1, Math.max(0, n));

// One drag per press: the pointer is captured, so moving off the area keeps steering it.
const dragWith = (onMove: (x: number, y: number) => void) => (event: PointerEvent<HTMLDivElement>) => {
  const area = event.currentTarget;
  area.setPointerCapture(event.pointerId);
  const at = (e: { clientX: number; clientY: number }) => {
    const box = area.getBoundingClientRect();
    onMove(clamp((e.clientX - box.left) / box.width), clamp((e.clientY - box.top) / box.height));
  };
  at(event);
  const move = (e: globalThis.PointerEvent) => at(e);
  area.addEventListener("pointermove", move);
  area.addEventListener("lostpointercapture", () => area.removeEventListener("pointermove", move), { once: true });
};

const arrows = (event: KeyboardEvent, step: (dx: number, dy: number) => void) => {
  const delta: Record<string, [number, number]> = { ArrowLeft: [-1, 0], ArrowRight: [1, 0], ArrowUp: [0, -1], ArrowDown: [0, 1] };
  const move = delta[event.key];
  if (!move) return;
  event.preventDefault();
  step(move[0] * (event.shiftKey ? 10 : 1), move[1] * (event.shiftKey ? 10 : 1));
};

const thumb = "pointer-events-none absolute size-3.5 -translate-x-1/2 -translate-y-1/2 rounded-full border-2 border-white shadow-[0_0_0_1px_rgb(0_0_0/0.35)]";

interface ColorPanelProps {
  value: string;
  // Every move of a drag, for a live preview.
  onChange: (color: string) => void;
  // A finished choice (a quick hue or No color), so a popover can close on it.
  onPick?: (color: string) => void;
  allowNone?: boolean;
}

// An owner's color: a saturation and brightness square, a hue strip, a hex field, and quick hues.
export const ColorPanel = ({ value, onChange, onPick = onChange, allowNone = true }: ColorPanelProps) => {
  const hex = NAMED_HEX[value] ?? value;
  const [hsv, setHsv] = useState<Hsv>(() => (HEX_COLOR.test(hex) ? hexToHsv(hex) : [220, 0.75, 0.95]));
  const [text, setText] = useState(hex);
  const [h, s, v] = hsv;

  const steer = (next: Hsv) => {
    setHsv(next);
    setText(hsvToHex(next));
    onChange(hsvToHex(next));
  };
  const pick = (color: string) => {
    if (color) setHsv(hexToHsv(color, h));
    setText(color);
    onPick(color);
  };

  return (
    <div className="flex w-56 flex-col gap-2.5">
      <div
        role="slider"
        tabIndex={0}
        aria-label="Saturation and brightness"
        aria-valuetext={`${Math.round(s * 100)}% saturation, ${Math.round(v * 100)}% brightness`}
        className="relative h-32 cursor-crosshair touch-none rounded-md focus-visible:ring-[3px] focus-visible:ring-ring/50 focus-visible:outline-none"
        style={{ background: `linear-gradient(to top, #000, transparent), linear-gradient(to right, #fff, ${hsvToHex([h, 1, 1])})` }}
        onPointerDown={dragWith((x, y) => steer([h, x, 1 - y]))}
        onKeyDown={(event) => arrows(event, (dx, dy) => steer([h, clamp(s + dx / 100), clamp(v - dy / 100)]))}
      >
        <span className={thumb} style={{ left: `${s * 100}%`, top: `${(1 - v) * 100}%`, backgroundColor: hsvToHex(hsv) }} />
      </div>
      <div
        role="slider"
        tabIndex={0}
        aria-label="Hue"
        aria-valuemin={0}
        aria-valuemax={359}
        aria-valuenow={Math.round(h)}
        className="relative h-3 cursor-pointer touch-none rounded-full focus-visible:ring-[3px] focus-visible:ring-ring/50 focus-visible:outline-none"
        style={{ background: HUE_STRIP }}
        onPointerDown={dragWith((x) => steer([x * 359, s, v]))}
        onKeyDown={(event) => arrows(event, (dx) => steer([Math.min(359, Math.max(0, h + dx)), s, v]))}
      >
        <span className={cn(thumb, "top-1/2")} style={{ left: `${(h / 359) * 100}%`, backgroundColor: hsvToHex([h, 1, 1]) }} />
      </div>
      <input
        aria-label="Hex color"
        spellCheck={false}
        autoComplete="off"
        className="h-7 w-full rounded-md border border-input bg-transparent px-2 font-mono text-xs uppercase outline-none focus-visible:border-ring dark:bg-input/20"
        placeholder="#000000"
        value={text}
        onChange={(event) => {
          const next = event.target.value.trim();
          setText(next);
          if (HEX_COLOR.test(next)) steer(hexToHsv(next.toLowerCase(), h));
        }}
      />
      <div role="radiogroup" aria-label="Quick colors" className="grid grid-cols-9 gap-1">
        {allowNone && (
          <button type="button" role="radio" aria-checked={value === ""} aria-label="No color" className={swatchClass(value === "")} onClick={() => pick("")}>
            <span className="size-full rounded-full border border-dashed border-muted-foreground" aria-hidden />
          </button>
        )}
        {QUICK_HUES.map((color) => (
          <button key={color} type="button" role="radio" aria-checked={hex === color} aria-label={color} className={swatchClass(hex === color)} onClick={() => pick(color)}>
            <span className="size-full rounded-full" style={{ backgroundColor: color }} aria-hidden />
          </button>
        ))}
      </div>
    </div>
  );
};

const swatchClass = (active: boolean) =>
  cn(
    "flex size-6 items-center justify-center rounded-full border-2 p-0.5 transition-colors duration-150 ease-standard focus-visible:border-ring focus-visible:outline-none",
    active ? "border-foreground/70" : "border-transparent hover:border-border",
  );
