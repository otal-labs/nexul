import type { Ref } from "react";

import { MotionRow } from "@/components/MotionRow";
import { PermissionsForm, PermissionsFormSchema, type PermissionsFormData } from "@/components/access/PermissionsForm";
import { PlayTemplateLine } from "@/components/play/PlayTemplateLine";
import { RowActionsMenu, type RowAction } from "@/components/settings/RowActionsMenu";
import { useCloneTemplateDialog } from "@/hooks/useCloneTemplateDialog";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useFormDialog } from "@/hooks/useFormDialog";
import { useDeletePlay } from "@/hooks/PlayHooks";
import { useFetchGrants } from "@/hooks/PermissionHooks";
import { PLAY_STAGE_LABELS, PLAY_TYPE_LABELS, type Play } from "@/models/Play";

interface PlayRowProps {
  ref?: Ref<HTMLLIElement>;
  // Its place in the list, so the rows after a removed one glide up.
  index: number;
  play: Play;
  workspaceId: string;
  canWrite: boolean;
  canDelete: boolean;
  onEdit: (play: Play) => void;
}

// canWrite gates "Exclude users" too: managing a play's plays:run exclusions takes the same plays:write
// bit as editing the play itself (ticket 21), so no separate permission wire is needed.
export const PlayRow = ({ ref, index, play, workspaceId, canWrite, canDelete, onEdit }: PlayRowProps) => {
  const deletePlay = useDeletePlay(workspaceId);
  const { open: openExclusions } = useFormDialog();
  const { open: confirm } = useConfirmationDialog();
  const openClone = useCloneTemplateDialog();
  const { data: grants } = useFetchGrants("play", play.id, canWrite);
  const excludedCount = (grants ?? []).filter((g) => g.deny.includes("plays:run")).length;

  const openExclusionsDialog = () =>
    openExclusions<PermissionsFormData>({
      title: "Exclude users",
      schema: PermissionsFormSchema,
      okLabel: "Apply",
      form: <PermissionsForm resourceType="play" resourceIds={[play.id]} />,
    });

  const remove = async () => {
    const ok = await confirm({
      title: `Delete ${play.label}?`,
      message: "Its button goes from every ticket, doc and Interview page. Runs it already made keep their trails.",
      confirmLabel: "Delete play",
    });
    if (ok) deletePlay.mutate(play.id);
  };

  const at = { scope: "workspace" as const, workspace_id: play.workspace_id };
  const actions: RowAction[] = [
    ...(canWrite ? [{ label: "Edit", onSelect: () => onEdit(play) }] : []),
    ...(canWrite ? [{ label: "Exclude users…", onSelect: () => void openExclusionsDialog() }] : []),
    ...(play.builtin_key
      ? [{ label: "Clone to…", onSelect: () => void openClone({ kind: "play_instructions", key: play.builtin_key, name: play.label, from: at }) }]
      : []),
    ...(canDelete ? [{ label: "Delete", destructive: true, onSelect: () => void remove() }] : []),
  ];

  return (
    <MotionRow ref={ref} index={index} className="flex items-start gap-3 bg-card px-4 py-3">
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
      {actions.length > 0 && <RowActionsMenu subject={play.label} actions={actions} />}
    </MotionRow>
  );
};
