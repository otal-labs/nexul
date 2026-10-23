import type { useSortable } from "@dnd-kit/sortable";
import { GripVertical } from "lucide-react";

interface RowGripProps {
  name: string;
  sortable: Pick<ReturnType<typeof useSortable>, "setActivatorNodeRef" | "attributes" | "listeners">;
}

// The handle that drags a row of a ranked list; the sortable's keyboard sensor moves it without a mouse.
export const RowGrip = ({ name, sortable: { setActivatorNodeRef, attributes, listeners } }: RowGripProps) => (
  <button
    type="button"
    ref={setActivatorNodeRef}
    {...attributes}
    {...listeners}
    aria-label={`Reorder ${name}`}
    className="-ml-1 flex size-6 shrink-0 cursor-grab touch-none items-center justify-center rounded-md text-muted-foreground/70 transition-colors duration-150 ease-standard hover:text-foreground focus-visible:outline-2 focus-visible:outline-focus active:cursor-grabbing"
  >
    <GripVertical className="size-4" aria-hidden />
  </button>
);
