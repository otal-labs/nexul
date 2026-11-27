import type { RowMenu } from "@/components/listpane/rowMenu";
import { useDeletePlay } from "@/hooks/PlayHooks";
import { useActionState } from "@/hooks/useActionState";
import { useCloneTemplateDialog } from "@/hooks/useCloneTemplateDialog";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { usePermissionsDialog } from "@/hooks/usePermissionsDialog";
import { usePlayDialogs } from "@/hooks/usePlayDialogs";
import { useRowGates } from "@/hooks/useRowGates";
import type { Play } from "@/models/Play";

// A play's row menu, shared by the Agent page's rows and the ticket rail; only a caller that can mount the History dialog passes onHistory. Permissions… follows plays:write too: a play's rules are written with the same bit as the play itself.
export const usePlayMenu = (play: Play, onHistory?: () => void): RowMenu => {
  const { offered, gates, deleteGate } = useRowGates({
    onEdit: { permission: "plays:write", verb: "edit plays" },
    onHistory: { permission: "plays:read", verb: "read play history" },
    onDelete: { permission: "plays:delete" },
    // A built-in clones its text onto a template; a play a person made clones into a new play.
    ...(play.builtin_key ? {} : { onClone: { permission: "plays:write", verb: "create plays" } }),
  });
  const canWrite = useActionState({ permission: "plays:write" }).state === "live";
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
    deletePlay.mutate(play.id);
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
    onHistory: onHistory && offered("onHistory") ? onHistory : undefined,
    permissions: canWrite ? { entity, onOpen: () => void openPermissions(entity) } : undefined,
    onDelete: offered("onDelete") ? () => void remove() : undefined,
  };
};
