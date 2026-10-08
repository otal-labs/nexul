import { CornerDownLeftIcon } from "lucide-react";

import { commandOptionId, type CommandItem } from "@/models/Command";
import { cn } from "@/lib/utils";

interface CommandPaletteOptionProps {
  item: CommandItem;
  index: number;
  active: boolean;
  onHover: (index: number) => void;
  onSelect: (item: CommandItem) => void;
}

// The active row carries the selection marks of the sidebar's active item: the lift, a brand edge and a brand icon. Its
// text takes accent-foreground, since a palette's accent may be a colour (Brutalism's yellow).
export const CommandPaletteOption = ({ item, index, active, onHover, onSelect }: CommandPaletteOptionProps) => {
  const Icon = item.icon;
  return (
    <div
      id={commandOptionId(index)}
      role="option"
      aria-selected={active}
      data-index={index}
      title={item.label}
      onPointerMove={() => onHover(index)}
      onClick={() => onSelect(item)}
      className={cn("relative flex h-9 cursor-default items-center gap-3 rounded-md px-3 text-sm select-none", active && "bg-accent text-accent-foreground")}
    >
      {active && <span aria-hidden className="absolute inset-y-2 left-0 w-0.5 rounded-full bg-brand" />}
      <Icon className={cn("size-4 shrink-0 text-muted-foreground", active && "text-brand")} />
      <span dir="auto" className="min-w-0 flex-1 truncate text-start">
        {item.label}
      </span>
      {item.hint && <span className={cn("shrink-0 font-mono text-xs tabular-nums", active ? "opacity-70" : "text-muted-foreground")}>{item.hint}</span>}
      <CornerDownLeftIcon aria-hidden className={cn("size-3.5 shrink-0 opacity-70", !active && "invisible")} />
    </div>
  );
};
