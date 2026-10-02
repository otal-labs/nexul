import { useSearchParams } from "react-router";

import { NOTE_VARIANTS, type NoteVariant } from "@/components/note/noteVariants";
import { cn } from "@/lib/utils";

interface NoteVariantSwitcherProps {
  current: NoteVariant;
  reader: boolean;
}

const chipClass = "h-7 rounded-full px-2 font-mono text-xs transition-colors duration-150 ease-standard";
const idleClass = "text-muted-foreground hover:bg-accent hover:text-foreground";

// Prototype only: flips ?note= and ?reader= in place, stacked above the thread layout pill.
export const NoteVariantSwitcher = ({ current, reader }: NoteVariantSwitcherProps) => {
  const [params, setParams] = useSearchParams();
  const set = (key: string, value: string | null) => {
    const next = new URLSearchParams(params);
    if (value === null) next.delete(key);
    if (value !== null) next.set(key, value);
    setParams(next, { replace: true });
  };

  return (
    <div className="fixed right-3 bottom-14 z-50 flex max-w-[calc(100vw-1.5rem)] items-center gap-2 rounded-full border border-border bg-popover py-1 pr-3 pl-1 shadow-elevated">
      <span className="pl-2 font-mono text-[11px] text-muted-foreground uppercase">Note</span>
      <div className="flex gap-0.5" role="group" aria-label="Note layout">
        {Object.values(NOTE_VARIANTS).map((variant) => (
          <button
            key={variant.key}
            type="button"
            title={variant.label}
            aria-pressed={variant.key === current.key}
            onClick={() => set("note", variant.key)}
            className={cn(chipClass, "w-7 uppercase", variant.key === current.key ? "bg-primary text-primary-foreground" : idleClass)}
          >
            {variant.key}
          </button>
        ))}
      </div>
      <button
        type="button"
        aria-pressed={reader}
        onClick={() => set("reader", reader ? null : "1")}
        className={cn(chipClass, reader ? "bg-primary text-primary-foreground" : idleClass)}
      >
        reader
      </button>
      <span className="hidden truncate text-xs text-muted-foreground md:inline">{current.label.slice(3)}</span>
    </div>
  );
};
