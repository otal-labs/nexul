import { useId } from "react";

import { CommandPaletteOption } from "@/components/command/CommandPaletteOption";
import { microheaderClass } from "@/components/Microheader";
import type { CommandGroup, CommandItem } from "@/models/Command";

interface CommandPaletteGroupProps {
  group: CommandGroup;
  // The flat index of the group's first item, so the keyboard walks every group as one list.
  start: number;
  activeIndex: number;
  onHover: (index: number) => void;
  onSelect: (item: CommandItem) => void;
}

export const CommandPaletteGroup = ({ group, start, activeIndex, onHover, onSelect }: CommandPaletteGroupProps) => {
  const headingId = useId();
  return (
    <div role="group" aria-labelledby={headingId} className="pb-1">
      <div id={headingId} className={`${microheaderClass} px-3 pt-3 pb-1.5`}>
        {group.heading}
      </div>
      {group.items.map((item, i) => (
        <CommandPaletteOption
          key={item.id}
          item={item}
          index={start + i}
          active={start + i === activeIndex}
          onHover={onHover}
          onSelect={onSelect}
        />
      ))}
    </div>
  );
};
