import { useNavigate } from "react-router";

import { CloneDocForm } from "@/components/doc/CloneDocForm";
import type { RowMenu } from "@/components/listpane/rowMenu";
import { useFetchDocFolders, useMoveDoc } from "@/hooks/DocFolderHooks";
import { useArchiveDoc, useDeleteDoc, useRestoreDoc, useSetDocLocked } from "@/hooks/DocHooks";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useDocPins } from "@/hooks/useDocPins";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useFormDialog } from "@/hooks/useFormDialog";
import { usePermissionsDialog } from "@/hooks/usePermissionsDialog";
import { useRowGates } from "@/hooks/useRowGates";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { CloneDocFormSchema, type CloneDocFormData, type DocListItem } from "@/models/Doc";
import { docPath, projectTokenById } from "@/models/Project";

// A doc row's menu: an action the viewer lacks is locked or left out (ADR 0152), and the server still checks the doc itself.
export const useDocRowActions = (doc: DocListItem, selected: boolean): RowMenu => {
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  const { pinnedIds, toggle } = useDocPins();
  const projectId = doc.project_id;
  const { offered, gates, deleteGate } = useRowGates({
    lock: { permission: "docs:lock", projectId, verb: doc.locked ? "unlock docs" : "lock docs" },
    archive: { permission: "docs:write", projectId, verb: doc.archived ? "restore docs" : "archive docs" },
    moveTo: { permission: "docs:write", projectId, verb: "move docs" },
    onClone: { permission: "docs:clone", projectId },
    onDelete: { permission: "docs:delete", projectId },
  });
  const { open: openForm } = useFormDialog();
  const { open: confirm } = useConfirmationDialog();
  const deleteDoc = useDeleteDoc();
  const setLocked = useSetDocLocked();
  const archiveDoc = useArchiveDoc();
  const restoreDoc = useRestoreDoc();
  const { data: projects } = useFetchProjects();
  const { data: folders } = useFetchDocFolders(doc.project_id);
  const moveDoc = useMoveDoc();
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const openPermissions = usePermissionsDialog();
  const entity = { type: "doc" as const, id: doc.id, name: doc.title, workspaceId, projectId: doc.project_id, folderId: doc.folder_id };

  // The dialog renders outside the router, so the move to the copy happens here once it resolves.
  const clone = async () => {
    const result = await openForm<CloneDocFormData>({
      title: "Clone to…",
      schema: CloneDocFormSchema,
      okLabel: "Clone doc",
      form: <CloneDocForm docId={doc.id} />,
      formOptions: { defaultValues: { project_id: doc.project_id } },
    });
    const cloned = result.data as (CloneDocFormData & { id?: string }) | null;
    if (cloned?.id) void navigate(wsPath(docPath(projectTokenById(projects ?? [], cloned.project_id), cloned.id)));
  };

  const remove = async () => {
    const ok = await confirm({ title: "Delete doc?", message: `"${doc.title}" and its history are deleted for good.`, confirmLabel: "Delete doc" });
    if (!ok) return;
    deleteDoc.mutate(doc.id, {
      onSuccess: () => {
        if (selected) void navigate(wsPath("/docs"));
      },
    });
  };

  return {
    pin: { pinned: pinnedIds.includes(doc.id), onToggle: () => toggle(doc.id) },
    gates,
    deleteGate,
    lock: offered("lock") ? { locked: doc.locked, onToggle: () => setLocked.mutate({ id: doc.id, locked: !doc.locked }) } : undefined,
    archive: offered("archive") ? { archived: doc.archived, onToggle: () => (doc.archived ? restoreDoc : archiveDoc).mutate(doc.id) } : undefined,
    moveTo:
      offered("moveTo") && folders && folders.length > 1
        ? { label: "Move to folder", options: folders, currentId: doc.folder_id, onMove: (folderId: string) => moveDoc.mutate({ id: doc.id, folderId }) }
        : undefined,
    onClone: offered("onClone") ? () => void clone() : undefined,
    permissions: { entity, onOpen: () => void openPermissions(entity) },
    onDelete: offered("onDelete") ? () => void remove() : undefined,
  };
};
