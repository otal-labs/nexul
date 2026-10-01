import { ChevronRightIcon, FolderIcon, FolderOpenIcon, PlusIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { RowActions } from "@/components/listpane/RowActions";
import { useDocFolderActions } from "@/hooks/useDocFolderActions";
import type { DocFolder } from "@/models/DocFolder";
import { cn } from "@/lib/utils";

interface DocFolderHeaderProps {
  folder: DocFolder;
  total: number;
  open: boolean;
  onToggle: () => void;
}

// The group label grown into a folder row: the label toggles the folder, and + and … take the count's place on hover and focus.
export const DocFolderHeader = ({ folder, total, open, onToggle }: DocFolderHeaderProps) => {
  const { onNewDoc, onRename, onDelete } = useDocFolderActions(folder, total);
  const hasActions = !!onNewDoc || !!onRename;
  return (
    <div className="group/folder relative flex h-9 items-center gap-1 border-b border-border pr-2 pl-3">
      <button
        type="button"
        onClick={onToggle}
        aria-expanded={open}
        className="flex min-w-0 flex-1 items-center gap-1.5 self-stretch font-mono text-[11px] font-medium tracking-[0.08em] text-muted-foreground uppercase outline-none hover:text-foreground focus-visible:text-foreground"
      >
        <ChevronRightIcon className={cn("size-3.5 shrink-0 transition-transform duration-150 ease-standard", open && "rotate-90")} aria-hidden />
        {open && <FolderOpenIcon className="size-3.5 shrink-0" aria-hidden />}
        {!open && <FolderIcon className="size-3.5 shrink-0" aria-hidden />}
        <span className="truncate">{folder.name}</span>
        <span className={cn("ml-auto tabular-nums", hasActions && "group-focus-within/folder:hidden group-hover/folder:hidden group-has-[[data-state=open]]/folder:hidden")}>
          {total}
        </span>
      </button>
      {hasActions && (
        <div className="hidden shrink-0 items-center gap-0.5 group-focus-within/folder:flex group-hover/folder:flex group-has-[[data-state=open]]/folder:flex">
          {onNewDoc && (
            <Button variant="ghost" size="icon" className="size-6" aria-label={`New doc in ${folder.name}`} title="New doc" onClick={onNewDoc}>
              <PlusIcon className="size-3.5" aria-hidden />
            </Button>
          )}
          {onRename && <RowActions itemLabel={folder.name} onRename={onRename} onDelete={onDelete} />}
        </div>
      )}
    </div>
  );
};
