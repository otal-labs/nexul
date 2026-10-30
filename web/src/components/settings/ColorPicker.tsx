import { NAMED_HEX } from "@/components/board/ticketTypeColor";
import { ColorPanel } from "@/components/settings/ColorPanel";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";

interface ColorPickerProps {
  label: string;
  value: string;
  onChange: (color: string) => void;
}

// A form's color field: the swatch and its hex, opening the color panel.
export const ColorPicker = ({ label, value, onChange }: ColorPickerProps) => {
  const hex = NAMED_HEX[value] ?? value;
  return (
    <Popover>
      <PopoverTrigger asChild>
        <button
          type="button"
          aria-label={label}
          className="inline-flex h-8 items-center gap-2 rounded-md border border-input px-2.5 text-xs transition-colors duration-150 ease-standard hover:bg-accent focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 focus-visible:outline-none dark:bg-input/20"
        >
          {hex && <span className="size-3.5 rounded-full" style={{ backgroundColor: hex }} aria-hidden />}
          {!hex && <span className="size-3.5 rounded-full border border-dashed border-muted-foreground" aria-hidden />}
          <span className="font-mono text-muted-foreground uppercase">{hex || "No color"}</span>
        </button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-auto p-2.5">
        <ColorPanel value={value} onChange={onChange} />
      </PopoverContent>
    </Popover>
  );
};
