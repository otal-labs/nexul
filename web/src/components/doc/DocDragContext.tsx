import { DndContext, DragOverlay, MouseSensor, pointerWithin, TouchSensor, useSensor, useSensors, type DragEndEvent, type DragStartEvent, type Modifier } from "@dnd-kit/core";
import { FileTextIcon, FolderIcon } from "lucide-react";
import { useState, type ReactNode } from "react";

import { docDrop, type DocDragItem } from "@/components/doc/docDrag";
import { useFetchDocFolders, useMoveDoc, useMoveDocFolder } from "@/hooks/DocFolderHooks";
import type { DocFolder } from "@/models/DocFolder";

// Module constants, so dnd-kit's context stays stable; 4px, so a slightly shaky click still opens the row.
const MOUSE = { activationConstraint: { distance: 4 } };
const TOUCH = { activationConstraint: { delay: 200, tolerance: 8 } };

// The chip hangs below and right of the pointer, so the folder row under it stays readable.
const besideCursor: Modifier = ({ activatorEvent, draggingNodeRect, transform }) => {
  const start = activatorEvent && ("touches" in activatorEvent ? (activatorEvent as TouchEvent).touches[0] : (activatorEvent as MouseEvent));
  if (!start || !draggingNodeRect) return transform;
  return { ...transform, x: transform.x + start.clientX - draggingNodeRect.left + 16, y: transform.y + start.clientY - draggingNodeRect.top + 16 };
};
const OVERLAY_MODIFIERS = [besideCursor];

// A drop back on the row it left would follow the link with a full page load: dnd-kit stops the click but not its default.
const swallowNextClick = () => {
  const stop = (event: MouseEvent) => event.preventDefault();
  window.addEventListener("click", stop, { capture: true, once: true });
  setTimeout(() => window.removeEventListener("click", stop, { capture: true }));
};

interface DocDragContextProps {
  projectId: string;
  children: ReactNode;
}

// Docs and folders drag onto a folder's row to move there; the row menus' Move to folder is the keyboard's way.
export const DocDragContext = ({ projectId, children }: DocDragContextProps) => {
  const sensors = useSensors(useSensor(MouseSensor, MOUSE), useSensor(TouchSensor, TOUCH));
  const { data: folders } = useFetchDocFolders(projectId);
  const moveDoc = useMoveDoc();
  const moveFolder = useMoveDocFolder();
  const [held, setHeld] = useState<DocDragItem | null>(null);

  const onDragStart = ({ active }: DragStartEvent) => setHeld((active.data.current as DocDragItem | undefined) ?? null);
  const onDragEnd = ({ active, over }: DragEndEvent) => {
    setHeld(null);
    swallowNextClick();
    const drop = docDrop(active.data.current as DocDragItem | undefined, over?.data.current as DocFolder | undefined, folders ?? []);
    if (drop?.kind === "doc") moveDoc.mutate({ id: drop.id, folderId: drop.folderId });
    if (drop?.kind === "folder") moveFolder.mutate({ id: drop.id, parentId: drop.parentId });
  };

  return (
    <DndContext sensors={sensors} collisionDetection={pointerWithin} onDragStart={onDragStart} onDragEnd={onDragEnd} onDragCancel={() => setHeld(null)}>
      {children}
      <DragOverlay dropAnimation={null} modifiers={OVERLAY_MODIFIERS}>
        {held && (
          <div className="flex w-fit max-w-64 cursor-grabbing items-center gap-1.5 rounded-md border border-border bg-card px-2.5 py-1.5 text-row text-foreground shadow-elevated">
            {held.kind === "doc" && <FileTextIcon className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />}
            {held.kind === "folder" && <FolderIcon className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />}
            <span className="truncate">{held.kind === "doc" ? held.doc.title : held.folder.name}</span>
          </div>
        )}
      </DragOverlay>
    </DndContext>
  );
};
