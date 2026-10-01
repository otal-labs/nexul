import { CalendarPlusIcon, PencilLineIcon } from "lucide-react";

import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { useDocSortStore } from "@/stores/docSortStore";
import type { DocSortField } from "@/models/Doc";

export const DocSortToggle = () => {
  const sortBy = useDocSortStore((s) => s.sortBy);
  const setSortBy = useDocSortStore((s) => s.setSortBy);
  return (
    <ToggleGroup
      type="single"
      variant="outline"
      size="sm"
      value={sortBy}
      // Radix reports "" when the active item is clicked again; keep the current sort.
      onValueChange={(value) => value && setSortBy(value as DocSortField)}
      aria-label="Sort docs by"
    >
      <ToggleGroupItem value="created_at" aria-label="Sort by created" title="Sort by created" className="px-2">
        <CalendarPlusIcon className="size-3.5" aria-hidden />
      </ToggleGroupItem>
      <ToggleGroupItem value="updated_at" aria-label="Sort by last edited" title="Sort by last edited" className="px-2">
        <PencilLineIcon className="size-3.5" aria-hidden />
      </ToggleGroupItem>
    </ToggleGroup>
  );
};
