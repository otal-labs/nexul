import { Controller, useForm } from "react-hook-form";

import { ColorPicker } from "@/components/settings/ColorPicker";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useRenameCategory } from "@/hooks/CategoryHooks";
import type { Category } from "@/models/Category";

interface CategoryEditFormProps {
  category: Category;
  onDone: () => void;
}

// The rename endpoint is full-replacement, so every save carries the color from the form, changed or not.
export const CategoryEditForm = ({ category, onDone }: CategoryEditFormProps) => {
  const renameCategory = useRenameCategory();
  const form = useForm<{ name: string; color: string }>({ defaultValues: { name: category.name, color: category.color } });

  const save = form.handleSubmit(async ({ name, color }) => {
    const trimmed = name.trim();
    if (trimmed && (trimmed !== category.name || color !== category.color)) {
      await renameCategory.mutateAsync({ id: category.id, name: trimmed, color });
    }
    onDone();
  });

  return (
    <form onSubmit={(event) => void save(event)} className="settle-in flex flex-col gap-2 px-3 py-2.5">
      <Controller
        control={form.control}
        name="name"
        render={({ field }) => (
          <Input
            aria-label="Category name"
            autoFocus
            {...field}
            onKeyDown={(event) => {
              if (event.key === "Escape") onDone();
            }}
          />
        )}
      />
      <div className="flex flex-wrap items-center justify-between gap-2">
        <Controller
          control={form.control}
          name="color"
          render={({ field }) => <ColorPicker label="Category color" value={field.value} onChange={field.onChange} />}
        />
        <div className="flex gap-1">
          <Button type="button" variant="ghost" size="sm" onClick={onDone}>
            Cancel
          </Button>
          <Button type="submit" size="sm" loading={renameCategory.isPending}>
            Save
          </Button>
        </div>
      </div>
    </form>
  );
};
