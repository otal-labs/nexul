import { useNavigate } from "react-router";

import { DocFolderNameForm } from "@/components/doc/DocFolderNameForm";
import { useDeleteDocFolder, useFetchDocFolders, useMoveDocFolder } from "@/hooks/DocFolderHooks";
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
import { folderOptions, subtreeIds } from "@/utils/DocFolderTreeUtility";

const moveLine = (n: number, to: string) => (n === 1 ? `1 doc moves to ${to}` : `${n} docs move to ${to}`);
const insideLine = (n: number) => (n === 1 ? "The folder inside goes with it" : `The ${n} folders inside go with it`);

// The top level is picked by this id, since a radio item needs a value; it moves the folder to parent_id "".
const topLevel = "top-level";

const deleteMessage = (total: number, inside: number, to: string) => {
  if (inside === 0 && total === 0) return "It's empty.";
  if (inside === 0) return `${moveLine(total, to)}. None are deleted.`;
  if (total === 0) return `${insideLine(inside)}. They hold no docs.`;
  return `${insideLine(inside)}, and ${moveLine(total, to)}. No doc is deleted.`;
};

// A folder row's actions, each locked or left out without its permission; the default folder offers no New doc, the header's + does.
export const useDocFolderActions = (folder: DocFolder, total: number) => {
  const projectId = folder.project_id;
  const { offered, gates, deleteGate } = useRowGates({
    onNewDoc: { permission: "docs:write", projectId, verb: "create docs" },
    onNewFolder: { permission: "docs:write", projectId, verb: "create folders" },
    moveTo: { permission: "docs:write", projectId, verb: "move folders" },
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
  const moveFolder = useMoveDocFolder();
  const { data: folders } = useFetchDocFolders(folder.project_id);
  const defaultName = folders?.find((f) => f.is_default)?.name ?? "the default folder";
  const parentName = folders?.find((f) => f.id === folder.parent_id)?.name;
  const inside = subtreeIds(folders ?? [], folder.id);
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

  const newFolder = () =>
    void openForm<DocFolderFormData>({
      title: `New folder in ${folder.name}`,
      schema: DocFolderFormSchema,
      okLabel: "Create folder",
      form: <DocFolderNameForm projectId={folder.project_id} parentId={folder.id} />,
      formOptions: { defaultValues: { name: "" } },
    });

  const remove = async () => {
    const message = deleteMessage(total, inside.size - 1, parentName ?? defaultName);
    const ok = await confirm({ title: `Delete ${folder.name}?`, message, confirmLabel: "Delete folder" });
    if (ok) deleteFolder.mutate(folder.id);
  };

  // A folder moves to the top or into any folder outside its own subtree.
  const moveTo = {
    label: "Move to folder",
    options: [{ id: topLevel, name: "Top level" }, ...folderOptions(folders ?? []).filter((option) => !inside.has(option.id))],
    currentId: folder.parent_id || topLevel,
    onMove: (id: string) => moveFolder.mutate({ id: folder.id, parentId: id === topLevel ? "" : id }),
  };

  return {
    gates,
    deleteGate,
    onNewDoc: !folder.is_default && offered("onNewDoc") ? () => void onNewDoc() : undefined,
    onNewFolder: offered("onNewFolder") ? newFolder : undefined,
    moveTo: !folder.is_default && offered("moveTo") ? moveTo : undefined,
    onRename: offered("onRename") ? rename : undefined,
    onDelete: !folder.is_default && offered("onDelete") ? () => void remove() : undefined,
    permissions: { entity, onOpen: () => void openPermissions(entity) },
  };
};
