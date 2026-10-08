import {
  closestCenter,
  DndContext,
  KeyboardSensor,
  MouseSensor,
  TouchSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
} from "@dnd-kit/core";
import { arrayMove, SortableContext, sortableKeyboardCoordinates, verticalListSortingStrategy } from "@dnd-kit/sortable";
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

  const sensors = useSensors(
    useSensor(MouseSensor, { activationConstraint: { distance: 2 } }),
    useSensor(TouchSensor, { activationConstraint: { delay: 200, tolerance: 8 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );

  const reorder = (from: number, to: number) => {
    const ids = (categories ?? []).map((c) => c.id);
    if (from === to || to < 0 || to >= ids.length) return;
    reorderCategories.mutate({ project_id: projectId, ids: arrayMove(ids, from, to) });
  };

  const onDragEnd = ({ active, over }: DragEndEvent) => {
    if (!over || !categories) return;
    const ids = categories.map((c) => c.id);
    reorder(ids.indexOf(String(active.id)), ids.indexOf(String(over.id)));
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
        <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={onDragEnd}>
          <SortableContext items={categories.map((c) => c.id)} strategy={verticalListSortingStrategy}>
            <ul className="mt-2 divide-y divide-border">
              {categories.map((category, index) => (
                <CategoryRow
                  key={category.id}
                  category={category}
                  count={countFor(category.id)}
                  first={index === 0}
                  last={index === categories.length - 1}
                  onMoveUp={() => reorder(index, index - 1)}
                  onMoveDown={() => reorder(index, index + 1)}
                />
              ))}
            </ul>
          </SortableContext>
        </DndContext>
      )}
    </div>
  );
};
