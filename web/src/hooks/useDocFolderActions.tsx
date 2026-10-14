import { useNavigate } from "react-router";

import { DocFolderNameForm } from "@/components/doc/DocFolderNameForm";
import { useDeleteDocFolder, useFetchDocFolders } from "@/hooks/DocFolderHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useOpenCreateDoc } from "@/hooks/useCreateDocDialog";
import { useFormDialog } from "@/hooks/useFormDialog";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { usePermissionsDialog } from "@/hooks/usePermissionsDialog";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { useRowGates } from "@/hooks/useRowGates";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { DocFolderFormSchema, type DocFolder, type DocFolderFormData } from "@/models/DocFolder";
import { docPath, projectTokenById } from "@/models/Project";

const moveLine = (n: number, to: string) => (n === 1 ? `1 doc moves to ${to}` : `${n} docs move to ${to}`);

// A folder row's actions, each locked or left out without its permission; the default folder offers no New doc, the header's + does.
export const useDocFolderActions = (folder: DocFolder, total: number) => {
  const projectId = folder.project_id;
  const { offered, gates, deleteGate } = useRowGates({
    onNewDoc: { permission: "docs:write", projectId, verb: "create docs" },
    onRename: { permission: "docs:write", projectId, verb: "rename folders" },
    onDelete: { permission: "docs:write", projectId, verb: "delete folders" },
  });
  const createDoc = useOpenCreateDoc(folder.project_id, folder.id);
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  const { data: projects } = useFetchProjects();
  const { open: openForm } = useFormDialog();
  const { open: confirm } = useConfirmationDialog();
  const deleteFolder = useDeleteDocFolder();
  const { data: folders } = useFetchDocFolders(folder.project_id);
  const defaultName = folders?.find((f) => f.is_default)?.name ?? "the default folder";
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const openPermissions = usePermissionsDialog();
  const entity = { type: "folder" as const, id: folder.id, name: folder.name, workspaceId, projectId: folder.project_id, folderId: folder.id };

  const onNewDoc = async () => {
    const id = await createDoc();
    if (id) void navigate(wsPath(docPath(projectTokenById(projects ?? [], folder.project_id), id)));
  };

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
    gates,
    deleteGate,
    onNewDoc: !folder.is_default && offered("onNewDoc") ? () => void onNewDoc() : undefined,
    onRename: offered("onRename") ? rename : undefined,
    onDelete: !folder.is_default && offered("onDelete") ? () => void remove() : undefined,
    permissions: { entity, onOpen: () => void openPermissions(entity) },
  };
};
