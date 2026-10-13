import { useState } from "react";

import { PlayTemplateLine } from "@/components/play/PlayTemplateLine";
import type { RowMenu } from "@/components/listpane/rowMenu";
import { RowContextMenu } from "@/components/listpane/RowContextMenu";
import { useCloneTemplateDialog } from "@/hooks/useCloneTemplateDialog";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useDeletePlay } from "@/hooks/PlayHooks";
import { useFetchGrants } from "@/hooks/PermissionHooks";
import { usePermissionsDialog } from "@/hooks/usePermissionsDialog";
import { leavingRowClass } from "@/hooks/useRowGlide";
import { cn } from "@/lib/utils";
import { PLAY_STAGE_LABELS, PLAY_TYPE_LABELS, type Play } from "@/models/Play";

interface PlayRowProps {
  // Called as the row starts to leave, so the list can glide the rows under it up once it is gone.
  onLeave?: () => void;
  play: Play;
  workspaceId: string;
  canWrite: boolean;
  canDelete: boolean;
  onEdit: (play: Play) => void;
}

// canWrite gates Permissions too: a play's rules are written with the same plays:write bit as the play itself.
export const PlayRow = ({ onLeave, play, workspaceId, canWrite, canDelete, onEdit }: PlayRowProps) => {
  const deletePlay = useDeletePlay(workspaceId);
  const [leaving, setLeaving] = useState(false);
  const openPermissions = usePermissionsDialog();
  const { open: confirm } = useConfirmationDialog();
  const openClone = useCloneTemplateDialog();
  const { data: grants } = useFetchGrants("play", play.id, canWrite);
  const excludedCount = (grants ?? []).filter((g) => g.deny.includes("plays:run")).length;

  const remove = async () => {
    const ok = await confirm({
      title: `Delete ${play.label}?`,
      message: "Its button goes from every ticket, doc and Interview page. Runs it already made keep their trails.",
      confirmLabel: "Delete play",
    });
    if (!ok) return;
    setLeaving(true);
    onLeave?.();
    deletePlay.mutate(play.id, { onError: () => setLeaving(false) });
  };

  const at = { scope: "workspace" as const, workspace_id: play.workspace_id };
  const entity = { type: "play" as const, id: play.id, name: play.label, workspaceId: play.workspace_id, projectId: "", folderId: "" };
  const menu: RowMenu = {
    onEdit: canWrite ? () => onEdit(play) : undefined,
    onClone: play.builtin_key ? () => void openClone({ kind: "play_instructions", key: play.builtin_key, name: play.label, from: at }) : undefined,
    permissions: canWrite ? { entity, onOpen: () => void openPermissions(entity) } : undefined,
    onDelete: canDelete ? () => void remove() : undefined,
  };

  return (
    <RowContextMenu menu={menu}>
      <li data-leaving={leaving || undefined} className={cn(leavingRowClass, "flex items-start gap-3 bg-card px-4 py-3")}>
        <div className="min-w-0 flex-1 space-y-1">
          <p className="line-clamp-2 text-sm font-medium break-words" title={play.label}>
            {play.label}
          </p>
          <p className="flex flex-wrap items-center gap-x-1.5 font-mono text-xs text-muted-foreground">
            <span>{PLAY_TYPE_LABELS[play.type]}</span>
            {play.type === "ticket" && play.show_when_stage && (
              <>
                <span aria-hidden>·</span>
                <span>{PLAY_STAGE_LABELS[play.show_when_stage]}</span>
              </>
            )}
            {excludedCount > 0 && (
              <>
                <span aria-hidden>·</span>
                <span>{excludedCount} excluded</span>
              </>
            )}
            {!play.enabled && (
              <>
                <span aria-hidden>·</span>
                <span className="inline-flex items-center gap-1 text-foreground/80">
                  <span className="size-1.5 rounded-full bg-muted-foreground" aria-hidden />
                  Disabled
                </span>
              </>
            )}
          </p>
          {play.description && <p className="line-clamp-2 text-xs text-muted-foreground">{play.description}</p>}
          {play.builtin_key && <PlayTemplateLine play={play} canWrite={canWrite} />}
        </div>
      </li>
    </RowContextMenu>
  );
};
