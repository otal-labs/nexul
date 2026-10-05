import { useState } from "react";

import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { menuItemClass, pillTriggerClass } from "@/components/ticket/ticketFormPillStyles";

interface PillPickerProps {
  label: string;
  items: { id: string; name: string }[];
  onPick: (id: string) => void;
}

// A create dialog header's pill that opens a one-pick list (project, folder).
export const PillPicker = ({ label, items, onPick }: PillPickerProps) => {
  const [open, setOpen] = useState(false);
  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <button type="button" className={pillTriggerClass}>
          {label}
        </button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-56 p-1">
        <div className="flex flex-col gap-0.5">
          {items.map((item) => (
            <button
              key={item.id}
              type="button"
              className={menuItemClass}
              onClick={() => {
                onPick(item.id);
                setOpen(false);
              }}
            >
              {item.name}
            </button>
          ))}
        </div>
      </PopoverContent>
    </Popover>
  );
};
