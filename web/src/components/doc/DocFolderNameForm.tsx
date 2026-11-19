import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { FormInput } from "@/components/FormInput";
import { useCreateDocFolder, useRenameDocFolder } from "@/hooks/DocFolderHooks";
import type { DocFolderFormData } from "@/models/DocFolder";

interface DocFolderNameFormProps {
  projectId: string;
  /** Set to rename this folder; omitted to create one in projectId. */
  folderId?: string | undefined;
  /** The folder a new one goes inside; omitted for the top level. */
  parentId?: string | undefined;
}

export const DocFolderNameForm = ({ projectId, folderId, parentId }: DocFolderNameFormProps) => {
  const { control, onSubmit } = useFormDialogContext<DocFolderFormData>();
  const createFolder = useCreateDocFolder();
  const renameFolder = useRenameDocFolder();

  onSubmit(async ({ name }) => {
    if (folderId) {
      await renameFolder.mutateAsync({ id: folderId, name });
      return { name };
    }
    await createFolder.mutateAsync({ projectId, parentId, name });
    return { name };
  });

  return <FormInput control={control} name="name" label="Name" placeholder="GetSource" autoFocus />;
};
