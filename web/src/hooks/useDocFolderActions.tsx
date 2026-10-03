import { DocFolderNameForm } from "@/components/doc/DocFolderNameForm";
import { useDeleteDocFolder, useFetchDocFolders } from "@/hooks/DocFolderHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useCreateDocDialog } from "@/hooks/useCreateDocDialog";
import { useFormDialog } from "@/hooks/useFormDialog";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import { DocFolderFormSchema, type DocFolder, type DocFolderFormData } from "@/models/DocFolder";

const moveLine = (n: number, to: string) => (n === 1 ? `1 doc moves to ${to}` : `${n} docs move to ${to}`);

// A folder row's New doc, Rename, and Delete, each undefined without docs:write; the default folder offers only Rename, the header's + files there.
export const useDocFolderActions = (folder: DocFolder, total: number) => {
  const canWrite = useHasPermission("docs:write");
  const onNewDoc = useCreateDocDialog(folder.project_id, folder.id);
  const { open: openForm } = useFormDialog();
  const { open: confirm } = useConfirmationDialog();
  const deleteFolder = useDeleteDocFolder();
  const { data: folders } = useFetchDocFolders(folder.project_id);
  const defaultName = folders?.find((f) => f.is_default)?.name ?? "the default folder";

  const rename = () =>
    void openForm<DocFolderFormData>({
      title: "Rename folder",
      schema: DocFolderFormSchema,
      okLabel: "Rename",
      form: <DocFolderNameForm projectId={folder.project_id} folderId={folder.id} />,
      formOptions: { defaultValues: { name: folder.name } },
    });

  const remove = async () => {
    const message = total === 0 ? "It holds no docs." : `${moveLine(total, defaultName)}; none is deleted.`;
    const ok = await confirm({ title: `Delete ${folder.name}?`, message, confirmLabel: "Delete" });
    if (ok) deleteFolder.mutate(folder.id);
  };

  return {
    onNewDoc: folder.is_default ? undefined : onNewDoc,
    onRename: canWrite ? rename : undefined,
    onDelete: canWrite && !folder.is_default ? () => void remove() : undefined,
  };
};
