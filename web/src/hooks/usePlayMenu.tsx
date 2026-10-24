import { useNavigate, useParams } from "react-router";

import type { RowMenu } from "@/components/listpane/rowMenu";
import { useDeletePlay } from "@/hooks/PlayHooks";
import { useActionState } from "@/hooks/useActionState";
import { useCloneTemplateDialog } from "@/hooks/useCloneTemplateDialog";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { usePermissionsDialog } from "@/hooks/usePermissionsDialog";
import { usePlayDialogs } from "@/hooks/usePlayDialogs";
import { useRowGates } from "@/hooks/useRowGates";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import type { Play } from "@/models/Play";

// A play's row menu, shared by its list row and its detail header. Permissions… follows plays:write too: a play's rules are written with the same bit as the play itself.
export const usePlayMenu = (play: Play): RowMenu => {
  const { offered, gates, deleteGate } = useRowGates({
    onEdit: { permission: "plays:write", verb: "edit plays" },
    onDelete: { permission: "plays:delete" },
    // A built-in clones its text onto a template; a play a person made clones into a new play.
    ...(play.builtin_key ? {} : { onClone: { permission: "plays:write", verb: "create plays" } }),
  });
  const canWrite = useActionState({ permission: "plays:write" }).state === "live";
  const { playId } = useParams();
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  const deletePlay = useDeletePlay(play.workspace_id);
  const openPermissions = usePermissionsDialog();
  const { open: confirm } = useConfirmationDialog();
  const openClone = useCloneTemplateDialog();
  const { edit, duplicate } = usePlayDialogs();

  const remove = async () => {
    const ok = await confirm({
      title: `Delete ${play.label}?`,
      message: "Its button goes from every ticket, doc and Interview page. Runs it already made keep their trails.",
      confirmLabel: "Delete play",
    });
    if (!ok) return;
    deletePlay.mutate(play.id, {
      onSuccess: () => {
        if (playId === play.id) void navigate(wsPath("/plays"));
      },
    });
  };

  const at = { scope: "workspace" as const, workspace_id: play.workspace_id };
  const clone = () => {
    if (play.builtin_key) return void openClone({ kind: "play_instructions", key: play.builtin_key, name: play.label, from: at });
    void duplicate(play);
  };
  const entity = { type: "play" as const, id: play.id, name: play.label, workspaceId: play.workspace_id, projectId: "", folderId: "" };
  return {
    gates,
    deleteGate,
    onEdit: offered("onEdit") ? () => void edit(play) : undefined,
    onClone: play.builtin_key || offered("onClone") ? clone : undefined,
    permissions: canWrite ? { entity, onOpen: () => void openPermissions(entity) } : undefined,
    onDelete: offered("onDelete") ? () => void remove() : undefined,
  };
};
