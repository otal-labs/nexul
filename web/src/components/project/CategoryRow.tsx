import { useSortable } from "@dnd-kit/sortable";
import { useState } from "react";
import { motion, useReducedMotion } from "motion/react";

import { CONFIGURABLE_COLOR_NAMES, HUE_DOT_CLASS, type ConfigurableColorName } from "@/components/board/ticketTypeColor";
import { CategoryEditForm } from "@/components/project/CategoryEditForm";
import { RowActionsMenu } from "@/components/settings/RowActionsMenu";
import { ColorDot } from "@/components/settings/SettingsListRow";
import { useDeleteCategory } from "@/hooks/CategoryHooks";
import { useActionState } from "@/hooks/useActionState";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { leavingRowClass } from "@/hooks/useRowGlide";
import { EASE_OUT, lastInputWasKeyboard, ROW_GLIDE } from "@/lib/motion";
import { cn } from "@/lib/utils";
import type { Category } from "@/models/Category";
import { countLabel } from "@/utils/CountUtility";

const isConfigurableColor = (color: string): color is ConfigurableColorName =>
  (CONFIGURABLE_COLOR_NAMES as readonly string[]).includes(color);

interface CategoryRowProps {
  category: Category;
  index: number;
  // The row a menu move was made on rides over the one it trades places with.
  lifted: boolean;
  count: number | undefined;
  first: boolean;
  last: boolean;
  onMoveUp: () => void;
  onMoveDown: () => void;
  // Called as the row starts to leave, so the list can glide the rows under it up once it is gone.
  onLeave?: () => void;
}

// Edit, reorder and delete sit in one menu so the row reads as dot, name, count and one affordance; the rest of the row is what drags.
export const CategoryRow = ({ category, index, lifted, count, first, last, onMoveUp, onMoveDown, onLeave }: CategoryRowProps) => {
  const reduced = useReducedMotion() ?? false;
  // A move made with the keyboard lands at once; a pointer's glides, so the eye follows the row to its new place.
  const glide = !reduced && !lastInputWasKeyboard();
  const deleteCategory = useDeleteCategory();
  const { open: confirm } = useConfirmationDialog();
  const [editing, setEditing] = useState(false);
  const [leaving, setLeaving] = useState(false);
  const draggable = useActionState({ permission: "projects:write", projectId: category.project_id }).state === "live";
  const { setNodeRef, setActivatorNodeRef, attributes, listeners, transform, transition, isDragging } = useSortable({
    id: category.id,
    disabled: editing || !draggable,
    transition: { duration: 200, easing: EASE_OUT },
  });

  const remove = async () => {
    const ok = await confirm({
      title: `Delete ${category.name}?`,
      message:
        count
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
      className={cn(
        leavingRowClass,
        "group/row relative flex min-h-11 flex-col justify-center bg-card px-2 before:pointer-events-none before:absolute before:inset-x-0 before:inset-y-1 before:rounded-md before:transition-colors before:duration-150 before:ease-standard hover:before:bg-accent/40",
        isDragging && "z-10 rounded-md shadow-elevated",
      )}
    >
      {editing && <CategoryEditForm category={category} onDone={() => setEditing(false)} />}
      {!editing && (
        <motion.div
          layout={glide ? "position" : false}
          layoutDependency={index}
          transition={{ layout: ROW_GLIDE }}
          className={cn("relative flex items-center gap-2 text-sm", lifted && "z-10")}
        >
          <div
            ref={setActivatorNodeRef}
            {...(draggable ? { ...attributes, ...listeners, "aria-label": `Reorder ${category.name}` } : {})}
            className={cn(
              "flex min-w-0 flex-1 items-center gap-2 self-stretch rounded-md",
              draggable && "cursor-grab select-none focus-visible:ring-[3px] focus-visible:ring-ring/50 focus-visible:outline-none active:cursor-grabbing",
            )}
          >
            <span className="flex size-4 shrink-0 items-center justify-center">
              <ColorDot className={isConfigurableColor(category.color) ? HUE_DOT_CLASS[category.color] : undefined} />
            </span>
            <span className="flex min-w-0 flex-1 items-baseline gap-2">
              <span className="min-w-0 truncate font-medium" title={category.name}>
                {category.name}
              </span>
              {count !== undefined && <span className="ml-auto shrink-0 pl-2 font-mono text-xs text-muted-foreground tabular-nums">{countLabel(count, "ticket")}</span>}
            </span>
          </div>
          <span className="flex shrink-0 [@media(hover:hover)]:opacity-0 [@media(hover:hover)]:group-focus-within/row:opacity-100 [@media(hover:hover)]:group-hover/row:opacity-100 [@media(hover:hover)]:has-data-[state=open]:opacity-100">
          <RowActionsMenu
            subject={category.name}
            projectId={category.project_id}
            actions={[
              { label: "Edit", permission: "projects:write", verb: "edit categories", onSelect: () => setEditing(true) },
              { label: "Move up", permission: "projects:write", verb: "reorder categories", disabled: first, onSelect: onMoveUp },
              { label: "Move down", permission: "projects:write", verb: "reorder categories", disabled: last, onSelect: onMoveDown },
              { label: "Delete", destructive: true, permission: "projects:delete", verb: "delete categories", onSelect: () => void remove() },
            ]}
          />
          </span>
        </motion.div>
      )}
    </li>
  );
};
