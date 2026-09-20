import { useState } from "react";

import { HUE_DOT_CLASS } from "@/components/board/ticketTypeColor";
import { ColorPicker } from "@/components/settings/ColorPicker";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { cn } from "@/lib/utils";

interface ColorSwatchButtonProps {
  label: string;
  value: string;
  onChange: (color: string) => void;
  allowNone?: boolean;
}

// One dot, not an inline swatch row — five hues per row breaks the color-as-signal ceiling at scale.
export const ColorSwatchButton = ({ label, value, onChange, allowNone = true }: ColorSwatchButtonProps) => {
  const [open, setOpen] = useState(false);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        aria-label={label}
        className="inline-flex size-6 items-center justify-center rounded-md transition-colors duration-150 ease-standard hover:bg-accent/60"
      >
        <span
          className={cn(
            "size-2.5 rounded-full",
            value && HUE_DOT_CLASS[value] ? HUE_DOT_CLASS[value] : "border border-dashed border-muted-foreground",
          )}
          aria-hidden
        />
      </PopoverTrigger>
      <PopoverContent align="end" className="w-auto p-1.5">
        <ColorPicker
          label={label}
          value={value}
          allowNone={allowNone}
          onChange={(color) => {
            setOpen(false);
            onChange(color);
          }}
        />
      </PopoverContent>
    </Popover>
  );
};
