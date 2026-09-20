import { FilterXIcon } from "lucide-react";

import { FilterChipRow } from "@/components/board/FilterChipRow";
import type { FilterRow } from "@/components/board/boardFilterChipBuilders";
import { Button } from "@/components/ui/button";
import { PopoverContent } from "@/components/ui/popover";

interface BoardFilterMenuProps {
  filterRows: FilterRow[];
  activeCount: number;
  onClear: () => void;
}

export const BoardFilterMenu = ({ filterRows, activeCount, onClear }: BoardFilterMenuProps) => (
  <PopoverContent className="flex w-auto max-w-[min(90vw,32rem)] flex-col gap-3">
    {filterRows.map((row) => (
      <FilterChipRow key={row.title} title={row.title} chips={row.chips} />
    ))}
    {activeCount > 0 && (
      // Divider/spacing lives on the wrapping div: pt-3 on the Button itself misaligned its hover fill.
      <div className="border-t border-border pt-3">
        <Button variant="ghost" size="sm" onClick={onClear} className="w-full justify-start text-xs">
          <FilterXIcon className="size-3.5" />
          Clear filter
        </Button>
      </div>
    )}
  </PopoverContent>
);
