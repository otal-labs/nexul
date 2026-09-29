import { PlusIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

interface ListPaneHeaderProps {
  title: string;
  count: number;
  newLabel: string;
  /** Omitted when the viewer may not create, which hides the button. */
  onNew?: (() => void) | undefined;
  search: string;
  onSearch: (value: string) => void;
}

export const ListPaneHeader = ({ title, count, newLabel, onNew, search, onSearch }: ListPaneHeaderProps) => (
  <div className="shrink-0">
    <div className="flex h-14 items-center justify-between border-b border-border pr-2 pl-4">
      <h1 className="flex items-baseline gap-2 text-sm font-semibold">
        {title}
        <span className="font-mono text-xs font-normal text-muted-foreground tabular-nums">{count}</span>
      </h1>
      {onNew && (
        <Button variant="ghost" size="icon" className="size-8" aria-label={newLabel} title={newLabel} onClick={onNew}>
          <PlusIcon className="size-4" aria-hidden />
        </Button>
      )}
    </div>
    <div className="px-3 pt-3 pb-2">
      <Input
        aria-label={`Search ${title.toLowerCase()}`}
        placeholder="Search…"
        value={search}
        onChange={(e) => onSearch(e.target.value)}
        className="h-8 text-xs"
      />
    </div>
  </div>
);
