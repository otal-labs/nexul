import { useNavigate } from "react-router";

import { CloneDocForm } from "@/components/doc/CloneDocForm";
import { useDeleteDoc } from "@/hooks/DocHooks";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useFormDialog } from "@/hooks/useFormDialog";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { CloneDocFormSchema, type CloneDocFormData, type DocListItem } from "@/models/Doc";
import { docPath, projectTokenById } from "@/models/Project";

// A row's Clone and Delete, each undefined when the viewer's role lacks it; the server still checks the doc itself.
export const useDocRowActions = (doc: DocListItem, selected: boolean) => {
  const navigate = useNavigate();
const wsPath = useWorkspacePath();
  const canClone = useHasPermission("docs:clone");
  const canDelete = useHasPermission("docs:delete");
  const { open: openForm } = useFormDialog();
  const { open: confirm } = useConfirmationDialog();
  const deleteDoc = useDeleteDoc();
  const { data: projects } = useFetchProjects();

  // The dialog renders outside the router, so the move to the copy happens here once it resolves.
  const clone = async () => {
    const result = await openForm<CloneDocFormData>({
      title: "Clone to…",
      schema: CloneDocFormSchema,
      okLabel: "Clone",
      form: <CloneDocForm docId={doc.id} />,
      formOptions: { defaultValues: { project_id: doc.project_id } },
    });
    const cloned = result.data as (CloneDocFormData & { id?: string }) | null;
    if (cloned?.id) void navigate(wsPath(docPath(projectTokenById(projects ?? [], cloned.project_id), cloned.id)));
  };

  const remove = async () => {
    const ok = await confirm({ title: "Delete doc?", message: `"${doc.title}" and its history are deleted for good.`, confirmLabel: "Delete" });
    if (!ok) return;
    deleteDoc.mutate(doc.id, {
      onSuccess: () => {
        if (selected) void navigate(wsPath("/docs"));
      },
    });
  };

  return { onClone: canClone ? () => void clone() : undefined, onDelete: canDelete ? () => void remove() : undefined };
};
