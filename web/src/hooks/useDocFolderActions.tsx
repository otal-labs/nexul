import { useNavigate } from "react-router";

import { DocFolderNameForm } from "@/components/doc/DocFolderNameForm";
import { useDeleteDocFolder, useFetchDocFolders } from "@/hooks/DocFolderHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useCreateDocDialog } from "@/hooks/useCreateDocDialog";
import { useFormDialog } from "@/hooks/useFormDialog";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import { DocFolderFormSchema, type DocFolder, type DocFolderFormData } from "@/models/DocFolder";
import { docPath, projectTokenById } from "@/models/Project";

const moveLine = (n: number, to: string) => (n === 1 ? `1 doc moves to ${to}` : `${n} docs move to ${to}`);

// A folder row's New doc, Rename, and Delete, each undefined without docs:write; the default folder offers only Rename, the header's + files there.
export const useDocFolderActions = (folder: DocFolder, total: number) => {
  const canWrite = useHasPermission("docs:write");
  const createDoc = useCreateDocDialog(folder.project_id, folder.id);
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  const { data: projects } = useFetchProjects();
  const { open: openForm } = useFormDialog();
  const { open: confirm } = useConfirmationDialog();
  const deleteFolder = useDeleteDocFolder();
  const { data: folders } = useFetchDocFolders(folder.project_id);
  const defaultName = folders?.find((f) => f.is_default)?.name ?? "the default folder";

  const onNewDoc =
    createDoc &&
    (async () => {
      const id = await createDoc();
      if (id) void navigate(wsPath(docPath(projectTokenById(projects ?? [], folder.project_id), id)));
    });

  const rename = () =>
    void openForm<DocFolderFormData>({
      title: "Rename folder",
      schema: DocFolderFormSchema,
      okLabel: "Rename",
      form: <DocFolderNameForm projectId={folder.project_id} folderId={folder.id} />,
      formOptions: { defaultValues: { name: folder.name } },
    });

  const remove = async () => {
    const message = total === 0 ? "It's empty." : `${moveLine(total, defaultName)}. None are deleted.`;
    const ok = await confirm({ title: `Delete ${folder.name}?`, message, confirmLabel: "Delete folder" });
    if (ok) deleteFolder.mutate(folder.id);
  };

  return {
    onNewDoc: folder.is_default ? undefined : onNewDoc,
    onRename: canWrite ? rename : undefined,
    onDelete: canWrite && !folder.is_default ? () => void remove() : undefined,
  };
};
