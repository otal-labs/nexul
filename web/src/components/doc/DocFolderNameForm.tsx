import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { FormInput } from "@/components/FormInput";
import { useCreateDocFolder, useRenameDocFolder } from "@/hooks/DocFolderHooks";
import type { DocFolderFormData } from "@/models/DocFolder";

interface DocFolderNameFormProps {
  projectId: string;
  /** Set to rename this folder; omitted to create one in projectId. */
  folderId?: string | undefined;
}

export const DocFolderNameForm = ({ projectId, folderId }: DocFolderNameFormProps) => {
  const { control, onSubmit } = useFormDialogContext<DocFolderFormData>();
  const createFolder = useCreateDocFolder();
  const renameFolder = useRenameDocFolder();

  onSubmit(async ({ name }) => {
    if (folderId) {
      await renameFolder.mutateAsync({ id: folderId, name });
      return { name };
    }
    await createFolder.mutateAsync({ projectId, name });
    return { name };
  });

  return <FormInput control={control} name="name" label="Name" placeholder="GetSource" autoFocus />;
};
