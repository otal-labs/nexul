import { useState } from "react";
import { Controller, useForm } from "react-hook-form";

import { CONFIGURABLE_COLOR_NAMES, HUE_DOT_CLASS, type ConfigurableColorName } from "@/components/board/ticketTypeColor";
import { ColorPicker } from "@/components/settings/ColorPicker";
import { RowActionsMenu } from "@/components/settings/RowActionsMenu";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useDeleteCategory, useRenameCategory } from "@/hooks/CategoryHooks";
import { cn } from "@/lib/utils";
import type { Category } from "@/models/Category";

const isConfigurableColor = (color: string): color is ConfigurableColorName =>
  (CONFIGURABLE_COLOR_NAMES as readonly string[]).includes(color);

interface CategoryRowProps {
  category: Category;
  count: number;
  first: boolean;
  last: boolean;
  onMoveUp: () => void;
  onMoveDown: () => void;
}

// Edit/reorder/delete collapse into the shared RowActionsMenu so the row reads as dot+name+count+one affordance.
export const CategoryRow = ({ category, count, first, last, onMoveUp, onMoveDown }: CategoryRowProps) => {
  const renameCategory = useRenameCategory();
  const deleteCategory = useDeleteCategory();
  const [editing, setEditing] = useState(false);
  const form = useForm<{ name: string; color: string }>({
    defaultValues: { name: category.name, color: category.color },
  });

  // The rename endpoint is full-replacement, so every save carries the color from the form, changed or not.
  const saveRename = () => {
    void form.handleSubmit(async ({ name, color }) => {
      const trimmed = name.trim();
      if (trimmed && (trimmed !== category.name || color !== category.color)) {
        await renameCategory.mutateAsync({ id: category.id, name: trimmed, color });
      }
      setEditing(false);
    })();
  };

  if (editing) {
    return (
      <li className="flex flex-col gap-1.5 py-1.5">
        <Controller
          control={form.control}
          name="name"
          render={({ field }) => (
            <Input
              id="category-name"
              className="flex-1"
              aria-label="Category name"
              autoFocus
              {...field}
              onKeyDown={(event) => {
                if (event.key === "Enter") saveRename();
                if (event.key === "Escape") {
                  form.reset({ name: category.name, color: category.color });
                  setEditing(false);
                }
              }}
            />
          )}
        />
        <div className="flex items-center justify-between gap-2">
          <Controller
            control={form.control}
            name="color"
            render={({ field }) => (
              <ColorPicker label="Category color" value={field.value} onChange={field.onChange} />
            )}
          />
          <Button variant="ghost" size="sm" onClick={saveRename}>
            Save
          </Button>
        </div>
      </li>
    );
  }

  return (
    <li className="-mx-2 flex items-center gap-2 rounded-sm px-2 py-1.5 text-sm transition-colors duration-[120ms] ease-standard hover:bg-accent/40">
      {/* Always occupies the dot column so names align whether or not a category has a color. */}
      <span
        className={cn(
          "size-2 shrink-0 rounded-full",
          isConfigurableColor(category.color) ? HUE_DOT_CLASS[category.color] : "bg-transparent",
        )}
        aria-hidden
      />
      <span className="flex-1 truncate">{category.name}</span>
      <span className="font-mono text-xs tabular-nums text-muted-foreground">{count}</span>
      <RowActionsMenu
        subject={category.name}
        actions={[
          { label: "Edit", onSelect: () => setEditing(true) },
          { label: "Move up", disabled: first, onSelect: onMoveUp },
          { label: "Move down", disabled: last, onSelect: onMoveDown },
          { label: "Delete", destructive: true, onSelect: () => deleteCategory.mutate(category.id) },
        ]}
      />
    </li>
  );
};
