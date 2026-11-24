import { useDraggable, useDroppable } from "@dnd-kit/core";
import { PlusIcon } from "lucide-react";
import { useCallback } from "react";

import { ActionButton } from "@/components/action/ActionButton";
import { docDrop, type DocDragItem } from "@/components/doc/docDrag";
import { FolderToggle } from "@/components/listpane/FolderToggle";
import { RowContextMenu } from "@/components/listpane/RowContextMenu";
import { hasOwnActions } from "@/components/listpane/rowMenu";
import { useFetchDocFolders } from "@/hooks/DocFolderHooks";
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
  const { data: folders } = useFetchDocFolders(folder.project_id);
  // The row is a drop target for docs and folders, and is itself held to move the folder, the default one excepted.
  const drop = useDroppable({ id: `folder:${folder.id}`, data: folder });
  const drag = useDraggable({ id: `folder-row:${folder.id}`, data: { kind: "folder", folder } satisfies DocDragItem, disabled: !menu.moveTo });
  const accepts = drop.isOver && !!docDrop(drop.active?.data.current as DocDragItem | undefined, folder, folders ?? []);
  const { setNodeRef: setDropNode } = drop;
  const { setNodeRef: setDragNode } = drag;
  const setRef = useCallback(
    (el: HTMLElement | null) => {
      setDropNode(el);
      setDragNode(el);
    },
    [setDropNode, setDragNode],
  );
  return (
    <RowContextMenu menu={menu} manages={manages}>
      <div
        ref={setRef}
        {...(menu.moveTo && drag.listeners)}
        className={cn("group/folder sticky-group-head flex items-center gap-1 pr-3 pl-3", accepts && "rounded-md ring-1 ring-ring ring-inset", drag.isDragging && "opacity-50")}
      >
        <FolderToggle
          name={folder.name}
          open={open}
          onToggle={onToggle}
          meta={total}
          metaClassName={cn(hasActions && "group-focus-within/folder:hidden group-hover/folder:hidden group-has-[[data-state=open]]/folder:hidden")}
        />
        {hasActions && (
          // -mr-2 sets the + glyph, centred in its 24px button, over the count's digits it replaces.
          <div className="-mr-2 hidden shrink-0 items-center group-focus-within/folder:flex group-hover/folder:flex group-has-[[data-state=open]]/folder:flex">
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
