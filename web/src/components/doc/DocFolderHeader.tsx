import { PlusIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { FolderToggle } from "@/components/listpane/FolderToggle";
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
      <FolderToggle
        name={folder.name}
        open={open}
        onToggle={onToggle}
        meta={total}
        metaClassName={cn(hasActions && "group-focus-within/folder:hidden group-hover/folder:hidden group-has-[[data-state=open]]/folder:hidden")}
      />
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
