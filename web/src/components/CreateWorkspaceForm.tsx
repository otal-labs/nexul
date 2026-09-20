import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { FormInput } from "@/components/FormInput";
import { useCreateWorkspace } from "@/hooks/WorkspaceHooks";
import type { SaveWorkspaceFormData } from "@/models/Workspace";
import { useWorkspaceStore } from "@/stores/workspaceStore";

export const CreateWorkspaceForm = () => {
  const { control, onSubmit } = useFormDialogContext<SaveWorkspaceFormData>();
  const createWorkspace = useCreateWorkspace();
  const selectWorkspace = useWorkspaceStore((s) => s.selectWorkspace);

  onSubmit(async ({ name }) => {
    const workspace = await createWorkspace.mutateAsync(name);
    selectWorkspace(workspace.id);
    return { name: workspace.name };
  });

  return (
    <FormInput
      control={control}
      name="name"
      label="Workspace name"
      placeholder="e.g. Shopkeepers"
      autoFocus
    />
  );
};
