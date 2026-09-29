import { CreateMemoryForm } from "@/components/memory/CreateMemoryForm";
import { useFormDialog } from "@/hooks/useFormDialog";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import { CreateMemoryFormSchema, emptyCreateMemoryForm, type CreateMemoryFormData } from "@/models/Memory";

// Opens New memory scoped to projectId; undefined without memories:write, so callers hide the button.
export const useCreateMemoryDialog = (projectId: string): (() => void) | undefined => {
  const canWrite = useHasPermission("memories:write");
  const { open } = useFormDialog();
  if (!canWrite) return undefined;
  return () =>
    void open<CreateMemoryFormData>({
      title: "New memory",
      schema: CreateMemoryFormSchema,
      okLabel: "Create",
      form: <CreateMemoryForm defaultProjectId={projectId} />,
      formOptions: { defaultValues: emptyCreateMemoryForm() },
    });
};
