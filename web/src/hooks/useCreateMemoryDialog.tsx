import { CreateMemoryForm } from "@/components/memory/CreateMemoryForm";
import { useFormDialog } from "@/hooks/useFormDialog";
import { CreateMemoryFormSchema, emptyCreateMemoryForm, type CreateMemoryFormData } from "@/models/Memory";

// Opens New memory scoped to projectId; the control that calls it gates itself.
export const useCreateMemoryDialog = (projectId: string): (() => void) => {
  const { open } = useFormDialog();
  return () =>
    void open<CreateMemoryFormData>({
      title: "New memory",
      schema: CreateMemoryFormSchema,
      okLabel: "Create memory",
      form: <CreateMemoryForm defaultProjectId={projectId} />,
      formOptions: { defaultValues: emptyCreateMemoryForm() },
    });
};
