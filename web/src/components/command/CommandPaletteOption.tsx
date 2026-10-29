import { Fragment } from "react";
import { CornerDownLeftIcon } from "lucide-react";

import { CommandAge, CommandChecked, CommandKey, CommandStatus } from "@/components/command/CommandPaletteMeta";
import { commandOptionId, type CommandItem } from "@/models/Command";
import { cn } from "@/lib/utils";

interface CommandPaletteOptionProps {
  item: CommandItem;
  index: number;
  active: boolean;
  onHover: (index: number) => void;
  onSelect: (item: CommandItem) => void;
}

const hasMeta = (item: CommandItem) => !!(item.hint || item.status || item.context || item.updatedAt || item.checked);

// The title over a meta line, or one 36px line when there is no meta. The active row takes the sidebar's selection marks;
// its text takes accent-foreground, since a palette's accent may be a colour (Brutalism's yellow).
export const CommandPaletteOption = ({ item, index, active, onHover, onSelect }: CommandPaletteOptionProps) => {
  const Icon = item.icon;
  const meta = hasMeta(item);
  return (
    <div
      id={commandOptionId(index)}
      role="option"
      aria-selected={active}
      data-index={index}
      title={item.label}
      onPointerMove={() => onHover(index)}
      onClick={() => onSelect(item)}
      className={cn(
        "relative flex items-center gap-3 rounded-md px-3 text-sm select-none",
        meta ? "h-12" : "h-9",
        active && "bg-accent text-accent-foreground",
      )}
    >
      {active && <span aria-hidden className="absolute inset-y-2 left-0 w-0.5 rounded-full bg-brand" />}
      <Icon className={cn("size-4 shrink-0 text-muted-foreground", active && "text-brand")} />
      <span className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span dir="auto" className="truncate text-start">
          {item.label}
        </span>
        {meta && <CommandMetaLine item={item} active={active} />}
      </span>
      <CornerDownLeftIcon aria-hidden className={cn("size-3.5 shrink-0 opacity-70", !active && "invisible")} />
    </div>
  );
};

// Parts are joined by a middot, except where a mark already separates them: the status dot, the current check.
const CommandMetaLine = ({ item, active }: { item: CommandItem; active: boolean }) => {
  const parts = [
    item.hint && { id: "key", marked: false, node: <CommandKey hint={item.hint} /> },
    item.status && { id: "status", marked: true, node: <CommandStatus status={item.status} /> },
    item.context && { id: "context", marked: false, node: <span className="truncate">{item.context}</span> },
    item.updatedAt && { id: "age", marked: false, node: <CommandAge at={item.updatedAt} /> },
    item.checked && { id: "checked", marked: true, node: <CommandChecked /> },
  ].filter((part) => !!part);
  return (
    <span className={cn("flex min-w-0 items-baseline text-xs", !active && "text-muted-foreground")}>
      {parts.map((part, i) => (
        <Fragment key={part.id}>
          {i > 0 && !part.marked && <span aria-hidden className="shrink-0 whitespace-pre"> · </span>}
          {i > 0 && part.marked && <span aria-hidden className="w-2 shrink-0" />}
          {part.node}
        </Fragment>
      ))}
    </span>
  );
};
