import { Controller } from "react-hook-form";

import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { FormInput } from "@/components/FormInput";
import { ColorPicker } from "@/components/settings/ColorPicker";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useCreateCategory } from "@/hooks/CategoryHooks";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import type { Category, SaveCategoryFormData } from "@/models/Category";

interface AddCategoryFormProps {
  // Fixed project context (ProjectCategories) hides the project picker; omitted (Board's toolbar shortcut) shows one.
  projectId?: string;
  onCreated?: (category: Category) => void;
}

// Shared with the Projects page (ADR 0006); style changes here affect both. Labeled fields per the
// locked Weekrise reference (2026-08-25, ~/Code/Images/weekrise-update-column-modal.jpg).
export const AddCategoryForm = ({ projectId, onCreated }: AddCategoryFormProps) => {
  const { control, onSubmit, formState } = useFormDialogContext<SaveCategoryFormData>();
  const createCategory = useCreateCategory();
  const { data: projects = [] } = useFetchProjects();

  onSubmit(async (input) => {
    const category = await createCategory.mutateAsync(input);
    onCreated?.(category);
    return { id: category.id, ...input };
  });

  const projectError = formState.errors.project_id;

  return (
    <div className="space-y-4">
      {!projectId && (
        <div className="space-y-2">
          <label htmlFor="category-project" className="text-sm font-medium">
            Project
          </label>
          <Controller
            control={control}
            name="project_id"
            render={({ field }) => (
              <Select value={field.value} onValueChange={field.onChange}>
                <SelectTrigger
                  id="category-project"
                  aria-label="Project"
                  aria-invalid={projectError != null}
                  onBlur={field.onBlur}
                >
                  <SelectValue placeholder="Select a project" />
                </SelectTrigger>
                <SelectContent>
                  {projects.map((project) => (
                    <SelectItem key={project.id} value={project.id}>
                      {project.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
          />
          {projectError && (
            <p role="alert" className="text-sm text-destructive">
              {projectError.message}
            </p>
          )}
        </div>
      )}
      <FormInput
        control={control}
        name="name"
        id="category-name"
        label="Name"
        placeholder="e.g. Sprint 1"
        autoFocus
        autoComplete="off"
        data-1p-ignore
        data-lpignore="true"
      />
      <div className="space-y-2">
        <span className="text-sm font-medium">Color</span>
        <div className="rounded-md border border-input px-2 py-1.5 dark:bg-input/20">
          <Controller
            control={control}
            name="color"
            render={({ field }) => (
              <ColorPicker label="Category color" value={field.value ?? ""} onChange={field.onChange} />
            )}
          />
        </div>
      </div>
    </div>
  );
};
