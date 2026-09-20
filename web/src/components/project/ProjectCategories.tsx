import { PlusIcon } from "lucide-react";

import { AddCategoryForm } from "@/components/project/AddCategoryForm";
import { CategoryRow } from "@/components/project/CategoryRow";
import { Button } from "@/components/ui/button";
import { useFetchProjectCategories, useReorderCategories } from "@/hooks/CategoryHooks";
import { useFormDialog } from "@/hooks/useFormDialog";
import { useFetchTicketsByProject } from "@/hooks/TicketHooks";
import { SaveCategoryFormSchema, type SaveCategoryFormData } from "@/models/Category";

interface ProjectCategoriesProps {
  projectId: string;
}

export const ProjectCategories = ({ projectId }: ProjectCategoriesProps) => {
  const { data: categories } = useFetchProjectCategories(projectId);
  const { data: tickets } = useFetchTicketsByProject(projectId);
  const reorderCategories = useReorderCategories();
  const { open: openAdd } = useFormDialog();

  const countFor = (categoryId: string) =>
    (tickets ?? []).filter((t) => t.category_id === categoryId).length;

  const move = (index: number, direction: -1 | 1) => {
    const ids = (categories ?? []).map((c) => c.id);
    if (ids.length === 0) return;
    const target = index + direction;
    const from = ids[index];
    const to = ids[target];
    if (target < 0 || target >= ids.length || from === undefined || to === undefined) return;
    ids[index] = to;
    ids[target] = from;
    void reorderCategories.mutateAsync({ project_id: projectId, ids });
  };

  const onAdd = async () => {
    await openAdd<SaveCategoryFormData>({
      title: "New category",
      schema: SaveCategoryFormSchema,
      okLabel: "Create category",
      form: <AddCategoryForm projectId={projectId} />,
      formOptions: { defaultValues: { project_id: projectId, name: "", color: "" } },
    });
  };

  return (
    <div className="mt-3">
      <div className="flex items-center justify-between gap-2">
        <span className="text-sm font-medium">Categories</span>
        <Button variant="ghost" size="sm" onClick={() => void onAdd()}>
          <PlusIcon className="size-3.5" />
          New category
        </Button>
      </div>
      {categories && categories.length === 0 && (
        <p className="mt-2 text-sm text-muted-foreground">
          No categories yet — the board groups tickets into swimlanes per category.
        </p>
      )}
      {categories && categories.length > 0 && (
        <ul className="mt-2 divide-y divide-border">
          {categories.map((category, index) => (
            <CategoryRow
              key={category.id}
              category={category}
              count={countFor(category.id)}
              first={index === 0}
              last={index === categories.length - 1}
              onMoveUp={() => move(index, -1)}
              onMoveDown={() => move(index, 1)}
            />
          ))}
        </ul>
      )}
    </div>
  );
};
