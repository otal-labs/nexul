import { CONFIGURABLE_COLOR_NAMES, HUE_DOT_CLASS } from "@/components/board/ticketTypeColor";
import { cn } from "@/lib/utils";

interface ColorPickerProps {
  label: string;
  value: string;
  onChange: (color: string) => void;
  /** Labels can't be cleared (no unset path); ticket types can, since color is nullable. */
  allowNone?: boolean;
}

// Selection uses a ring, not an underline — a border-bottom read as a rendering glitch.
const optionClass = (active: boolean) =>
  cn(
    "inline-flex size-7 shrink-0 items-center justify-center rounded-full border-2 transition-colors duration-150 ease-standard focus-visible:border-ring focus-visible:outline-none",
    active ? "border-foreground/60" : "border-transparent hover:border-border",
  );

// Shared 5-hue palette (ADR 0005); plain controlled props so rows can reuse it without a form lib.
export const ColorPicker = ({ label, value, onChange, allowNone = true }: ColorPickerProps) => (
  <div role="radiogroup" aria-label={label} className="flex flex-wrap items-center gap-1">
    {allowNone && (
      <button
        type="button"
        role="radio"
        aria-checked={value === ""}
        aria-label="No color"
        className={optionClass(value === "")}
        onClick={() => onChange("")}
      >
        <span className="size-4 rounded-full border border-dashed border-muted-foreground" aria-hidden />
      </button>
    )}
    {CONFIGURABLE_COLOR_NAMES.map((hue) => (
      <button
        key={hue}
        type="button"
        role="radio"
        aria-checked={value === hue}
        aria-label={hue}
        className={optionClass(value === hue)}
        onClick={() => onChange(hue)}
      >
        <span className={cn("size-4 rounded-full", HUE_DOT_CLASS[hue])} aria-hidden />
      </button>
    ))}
  </div>
);
