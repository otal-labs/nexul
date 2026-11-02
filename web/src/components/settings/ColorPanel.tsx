import { RadioGroup as RadioGroupPrimitive } from "radix-ui";
import { useId, useState, type KeyboardEvent, type PointerEvent } from "react";

import { NAMED_HEX } from "@/components/board/ticketTypeColor";
import { cn } from "@/lib/utils";
import { HEX_COLOR, hexToHsv, hsvToHex, type Hsv } from "@/lib/hsv";

// The person fill's 12 hues at chroma 0.11, lifted to lightness 0.6 so a type icon in one holds 3:1 on the dark card and on white.
const QUICK_COLORS = [
  { name: "Red", hex: "#b96558" },
  { name: "Orange", hex: "#b06f35" },
  { name: "Amber", hex: "#9a7d22" },
  { name: "Olive", hex: "#788a37" },
  { name: "Green", hex: "#4a925c" },
  { name: "Teal", hex: "#009583" },
  { name: "Cyan", hex: "#0091a4" },
  { name: "Blue", hex: "#3788bb" },
  { name: "Indigo", hex: "#677cc2" },
  { name: "Violet", hex: "#8a70b8" },
  { name: "Plum", hex: "#a5679f" },
  { name: "Rose", hex: "#b5627d" },
  { name: "Slate", hex: "#7a808d" },
];
const NONE = "none";

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
  // A finished choice (a quick color clicked, or Enter or Space on one), so a popover can close on it; arrowing through them only previews.
  onPick?: (color: string) => void;
  allowNone?: boolean;
}

// An owner's color: a saturation and brightness square, a hue strip, a hex field, and quick hues.
export const ColorPanel = ({ value, onChange, onPick = onChange, allowNone = true }: ColorPanelProps) => {
  const hex = NAMED_HEX[value] ?? value;
  const [hsv, setHsv] = useState<Hsv>(() => (HEX_COLOR.test(hex) ? hexToHsv(hex) : [220, 0.75, 0.95]));
  const [text, setText] = useState(hex);
  const [h, s, v] = hsv;
  const invalid = text !== "" && !HEX_COLOR.test(text);
  const errorId = useId();

  const steer = (next: Hsv) => {
    setHsv(next);
    setText(hsvToHex(next));
    onChange(hsvToHex(next));
  };
  const choose = (color: string, done: boolean) => {
    if (color) setHsv(hexToHsv(color, h));
    setText(color);
    (done ? onPick : onChange)(color);
  };
  const quickValue = (color: string) => (color === NONE ? "" : color);
  const pickQuick = (color: string) => choose(quickValue(color), true);

  return (
    <div className="flex w-56 flex-col gap-2.5">
      <div
        role="slider"
        tabIndex={0}
        aria-label="Saturation and brightness"
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={Math.round(v * 100)}
        aria-valuetext={`${Math.round(s * 100)}% saturation, ${Math.round(v * 100)}% brightness`}
        className="relative h-32 cursor-crosshair touch-none rounded-md"
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
        className="relative h-3 cursor-pointer touch-none rounded-full"
        style={{ background: HUE_STRIP }}
        onPointerDown={dragWith((x) => steer([x * 359, s, v]))}
        onKeyDown={(event) => arrows(event, (dx) => steer([Math.min(359, Math.max(0, h + dx)), s, v]))}
      >
        <span className={cn(thumb, "top-1/2")} style={{ left: `${(h / 359) * 100}%`, backgroundColor: hsvToHex([h, 1, 1]) }} />
      </div>
      <input
        aria-label="Hex color"
        aria-invalid={invalid}
        aria-describedby={invalid ? errorId : undefined}
        spellCheck={false}
        autoComplete="off"
        className="h-7 w-full rounded-md border border-input bg-transparent px-2 font-mono text-xs uppercase aria-invalid:border-destructive dark:bg-input/20"
        placeholder="#000000"
        value={text}
        onChange={(event) => {
          const next = event.target.value.trim();
          setText(next);
          if (HEX_COLOR.test(next)) steer(hexToHsv(next.toLowerCase(), h));
        }}
        onBlur={() => {
          if (!invalid) return;
          setText(value === "" ? "" : hsvToHex(hsv));
        }}
      />
      {invalid && (
        <p id={errorId} role="alert" className="-mt-1 text-xs text-destructive">
          Use six hex digits, like #3788BB
        </p>
      )}
      <RadioGroupPrimitive.Root
        aria-label="Quick colors"
        value={value === "" ? NONE : hex}
        onValueChange={(color) => choose(quickValue(color), false)}
        className="grid grid-cols-[repeat(7,auto)] justify-between gap-y-1.5"
      >
        {allowNone && <QuickColor value={NONE} name="No color" active={value === ""} onPick={pickQuick} />}
        {QUICK_COLORS.map((color) => (
          <QuickColor key={color.hex} value={color.hex} name={color.name} active={hex === color.hex} onPick={pickQuick} />
        ))}
      </RadioGroupPrimitive.Root>
    </div>
  );
};

interface QuickColorProps {
  value: string;
  name: string;
  active: boolean;
  onPick: (value: string) => void;
}

// Radix checks an item as the arrows reach it, through a click with no pointer behind it, so only a real click or Enter and Space pick.
const QuickColor = ({ value, name, active, onPick }: QuickColorProps) => (
  <RadioGroupPrimitive.Item
    value={value}
    aria-label={name}
    className={cn(
      "flex size-6 items-center justify-center rounded-full border-2 p-0.5 transition-colors duration-150 ease-standard",
      active ? "border-foreground/70" : "border-transparent hover:border-border",
    )}
    onClick={(event) => {
      if (event.detail > 0) onPick(value);
    }}
    onKeyDown={(event) => {
      if (event.key === "Enter" || event.key === " ") onPick(value);
    }}
  >
    {value === NONE && <span className="size-full rounded-full border border-dashed border-muted-foreground" aria-hidden />}
    {value !== NONE && <span className="size-full rounded-full" style={{ backgroundColor: value }} aria-hidden />}
  </RadioGroupPrimitive.Item>
);
