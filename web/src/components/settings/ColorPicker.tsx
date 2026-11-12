import { NAMED_HEX } from "@/components/board/ticketTypeColor";
import { ColorPanel } from "@/components/settings/ColorPanel";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";

interface ColorPickerProps {
  label: string;
  value: string;
  onChange: (color: string) => void;
  disabled?: boolean;
  /** What an empty value reads as, with the live accent as its swatch; without it an empty value is "No color". */
  emptyLabel?: string;
}

// A form's color field: the swatch and its hex, opening the color panel.
export const ColorPicker = ({ label, value, onChange, disabled = false, emptyLabel }: ColorPickerProps) => {
  const hex = NAMED_HEX[value] ?? value;
  return (
    <Popover>
      <PopoverTrigger asChild>
        <button
          type="button"
          aria-label={label}
          disabled={disabled}
          className="hit-area inline-flex h-8 items-center gap-2 rounded-md disabled:pointer-events-none disabled:text-muted-foreground disabled:opacity-50 border border-input px-2.5 text-xs transition-colors duration-150 ease-standard hover:bg-accent dark:bg-input/20"
        >
          {hex && <span className="size-3.5 rounded-full" style={{ backgroundColor: hex }} aria-hidden />}
          {!hex && emptyLabel && <span className="size-3.5 rounded-full bg-brand" aria-hidden />}
          {!hex && !emptyLabel && <span className="size-3.5 rounded-full border border-dashed border-muted-foreground" aria-hidden />}
          {hex && <span className="font-mono text-muted-foreground uppercase">{hex}</span>}
          {!hex && emptyLabel && <span className="text-muted-foreground">{emptyLabel}</span>}
          {!hex && !emptyLabel && <span className="font-mono text-muted-foreground uppercase">No color</span>}
        </button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-auto p-2.5">
        <ColorPanel value={value} onChange={onChange} />
      </PopoverContent>
    </Popover>
  );
};
