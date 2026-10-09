import { useSortable } from "@dnd-kit/sortable";
import { GripVerticalIcon } from "lucide-react";
import { useState } from "react";
import { motion, useReducedMotion } from "motion/react";

import { CONFIGURABLE_COLOR_NAMES, HUE_DOT_CLASS, type ConfigurableColorName } from "@/components/board/ticketTypeColor";
import { CategoryEditForm } from "@/components/project/CategoryEditForm";
import { RowActionsMenu } from "@/components/settings/RowActionsMenu";
import { useDeleteCategory } from "@/hooks/CategoryHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { leavingRowClass } from "@/hooks/useRowGlide";
import { EASE_OUT, lastInputWasKeyboard, ROW_GLIDE } from "@/lib/motion";
import { cn } from "@/lib/utils";
import type { Category } from "@/models/Category";

const isConfigurableColor = (color: string): color is ConfigurableColorName =>
  (CONFIGURABLE_COLOR_NAMES as readonly string[]).includes(color);

interface CategoryRowProps {
  category: Category;
  index: number;
  // The row a menu move was made on rides over the one it trades places with.
  lifted: boolean;
  count: number;
  first: boolean;
  last: boolean;
  onMoveUp: () => void;
  onMoveDown: () => void;
  // Called as the row starts to leave, so the list can glide the rows under it up once it is gone.
  onLeave?: () => void;
}

// Edit, reorder and delete sit in one menu so the row reads as grip, dot, name, count and one affordance.
export const CategoryRow = ({ category, index, lifted, count, first, last, onMoveUp, onMoveDown, onLeave }: CategoryRowProps) => {
  const reduced = useReducedMotion() ?? false;
  // A move made with the keyboard lands at once; a pointer's glides, so the eye follows the row to its new place.
  const glide = !reduced && !lastInputWasKeyboard();
  const deleteCategory = useDeleteCategory();
  const { open: confirm } = useConfirmationDialog();
  const [editing, setEditing] = useState(false);
  const [leaving, setLeaving] = useState(false);
  const { setNodeRef, setActivatorNodeRef, attributes, listeners, transform, transition, isDragging } = useSortable({
    id: category.id,
    disabled: editing,
    transition: { duration: 200, easing: EASE_OUT },
  });

  const remove = async () => {
    const ok = await confirm({
      title: `Delete ${category.name}?`,
      message:
        count > 0
          ? `Its swimlane goes. ${count === 1 ? "Its 1 ticket stays" : `Its ${count} tickets stay`} on the board without a category.`
          : "Its swimlane goes from the board.",
      confirmLabel: "Delete category",
    });
    if (!ok) return;
    setLeaving(true);
    onLeave?.();
    deleteCategory.mutate(category.id, { onError: () => setLeaving(false) });
  };

  return (
    <li
      ref={setNodeRef}
      style={{ transform: transform ? `translate3d(0, ${Math.round(transform.y)}px, 0)` : undefined, transition }}
      data-leaving={leaving || undefined}
      className={cn(leavingRowClass, "relative bg-card", isDragging && "z-10 rounded-md shadow-elevated")}
    >
      {editing && <CategoryEditForm category={category} onDone={() => setEditing(false)} />}
      {!editing && (
        <motion.div
          layout={glide ? "position" : false}
          layoutDependency={index}
          transition={{ layout: ROW_GLIDE }}
          className={cn(
            "relative flex items-center gap-2 bg-card px-3 py-2 text-sm transition-colors duration-150 ease-standard hover:bg-accent/40",
            lifted && "z-10",
          )}
        >
          <button
            ref={setActivatorNodeRef}
            type="button"
            aria-label={`Reorder ${category.name}`}
            className="-ml-1 cursor-grab touch-none rounded-md p-0.5 text-muted-foreground/60 hover:text-foreground focus-visible:ring-[3px] focus-visible:ring-ring/50 focus-visible:outline-none active:cursor-grabbing"
            {...attributes}
            {...listeners}
          >
            <GripVerticalIcon className="size-3.5" aria-hidden />
          </button>
          {/* Always holds the dot column so names line up with or without a color. */}
          <span
            className={cn("size-2 shrink-0 rounded-full", isConfigurableColor(category.color) ? HUE_DOT_CLASS[category.color] : "bg-transparent")}
            aria-hidden
          />
          <span className="min-w-0 flex-1 truncate" title={category.name}>
            {category.name}
          </span>
          <span className="font-mono text-xs tabular-nums text-muted-foreground" title={`${count} tickets`}>
            {count}
          </span>
          <RowActionsMenu
            subject={category.name}
            actions={[
              { label: "Edit", onSelect: () => setEditing(true) },
              { label: "Move up", disabled: first, onSelect: onMoveUp },
              { label: "Move down", disabled: last, onSelect: onMoveDown },
              { label: "Delete", destructive: true, onSelect: () => void remove() },
            ]}
          />
        </motion.div>
      )}
    </li>
  );
};
