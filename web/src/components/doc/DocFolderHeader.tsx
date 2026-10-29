import { PlusIcon } from "lucide-react";

import { ActionButton } from "@/components/action/ActionButton";
import { FolderToggle } from "@/components/listpane/FolderToggle";
import { RowContextMenu } from "@/components/listpane/RowContextMenu";
import { hasOwnActions } from "@/components/listpane/rowMenu";
import { useCanManageAccess } from "@/hooks/useCanManageAccess";
import { useDocFolderActions } from "@/hooks/useDocFolderActions";
import type { DocFolder } from "@/models/DocFolder";
import { cn } from "@/lib/utils";

interface DocFolderHeaderProps {
  folder: DocFolder;
  total: number;
  open: boolean;
  onToggle: () => void;
}

// The group label grown into a folder row: the label toggles the folder, and + takes the count's place on hover and focus.
export const DocFolderHeader = ({ folder, total, open, onToggle }: DocFolderHeaderProps) => {
  const menu = useDocFolderActions(folder, total);
  const hasActions = !!menu.onNewDoc;
  // Asked up front only where Permissions… could be the menu's one item, so whoever manages the folder can reach it.
  const manages = useCanManageAccess(hasOwnActions(menu) || menu.onDelete ? undefined : menu.permissions?.entity);
  return (
    <RowContextMenu menu={menu} manages={manages}>
      <div className="group/folder relative flex h-9 items-center gap-1 pr-3 pl-3">
        <FolderToggle
          name={folder.name}
          open={open}
          onToggle={onToggle}
          meta={total}
          metaClassName={cn(hasActions && "group-focus-within/folder:hidden group-hover/folder:hidden group-has-[[data-state=open]]/folder:hidden")}
        />
        {hasActions && (
          <div className="hidden shrink-0 items-center group-focus-within/folder:flex group-hover/folder:flex group-has-[[data-state=open]]/folder:flex">
            <ActionButton
              permission="docs:write"
              projectId={folder.project_id}
              verb="create docs"
              variant="ghost"
              size="icon"
              className="size-6"
              aria-label={`New doc in ${folder.name}`}
              title="New doc"
              icon={PlusIcon}
              onClick={menu.onNewDoc}
            />
          </div>
        )}
      </div>
    </RowContextMenu>
  );
};
