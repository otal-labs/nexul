import { ChevronDownIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useDocArchiveStore } from "@/stores/docArchiveStore";
import { useDocSortStore } from "@/stores/docSortStore";
import type { DocSortField } from "@/models/Doc";

const sortLabels: Record<DocSortField, string> = { updated_at: "Edited", created_at: "Created" };

// The sort and the archived view in one labelled control; the label names both, so an archived list never reads as empty.
export const DocListMenu = () => {
  const sortBy = useDocSortStore((s) => s.sortBy);
  const setSortBy = useDocSortStore((s) => s.setSortBy);
  const showArchived = useDocArchiveStore((s) => s.showArchived);
  const setShowArchived = useDocArchiveStore((s) => s.setShowArchived);
  const label = showArchived ? `Archived · ${sortLabels[sortBy]}` : sortLabels[sortBy];
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="sm" className="h-8 gap-1 px-2 text-xs text-muted-foreground" aria-label={`Sort and view: ${label}`}>
          {label}
          <ChevronDownIcon className="size-3.5" aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-48">
        <DropdownMenuLabel>Sort by</DropdownMenuLabel>
        <DropdownMenuRadioGroup value={sortBy} onValueChange={(value) => setSortBy(value as DocSortField)}>
          <DropdownMenuRadioItem value="updated_at">Last edited</DropdownMenuRadioItem>
          <DropdownMenuRadioItem value="created_at">Created</DropdownMenuRadioItem>
        </DropdownMenuRadioGroup>
        <DropdownMenuSeparator />
        <DropdownMenuCheckboxItem checked={showArchived} onCheckedChange={(checked) => setShowArchived(checked === true)}>
          Show archived
        </DropdownMenuCheckboxItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
