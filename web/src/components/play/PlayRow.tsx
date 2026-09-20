import { Trash2 } from "lucide-react";

import { PermissionsForm, PermissionsFormSchema, type PermissionsFormData } from "@/components/access/PermissionsForm";
import { ConfirmDestroyButton } from "@/components/settings/ConfirmDestroyButton";
import { NoFillBadge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useFormDialog } from "@/hooks/useFormDialog";
import { useDeletePlay } from "@/hooks/PlayHooks";
import { useFetchGrants } from "@/hooks/PermissionHooks";
import { PLAY_STAGE_LABELS, type Play } from "@/models/Play";

interface PlayRowProps {
  play: Play;
  workspaceId: string;
  canWrite: boolean;
  canDelete: boolean;
  onEdit: (play: Play) => void;
}

// canWrite gates "Exclude users" too: managing a play's plays:run exclusions takes the same plays:write
// bit as editing the play itself (ticket 21), so no separate permission wire is needed.
export const PlayRow = ({ play, workspaceId, canWrite, canDelete, onEdit }: PlayRowProps) => {
  const deletePlay = useDeletePlay(workspaceId);
  const { open: openExclusions } = useFormDialog();
  const { data: grants } = useFetchGrants("play", play.id, canWrite);
  const excludedCount = (grants ?? []).filter((g) => g.deny.includes("plays:run")).length;

  const openExclusionsDialog = () =>
    openExclusions<PermissionsFormData>({
      title: "Exclude users",
      schema: PermissionsFormSchema,
      okLabel: "Apply",
      form: <PermissionsForm resourceType="play" resourceIds={[play.id]} />,
    });

  return (
    <li className="flex flex-wrap items-center gap-3 bg-card px-3 py-3">
      <div className="flex min-w-0 flex-1 flex-col gap-1">
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-sm font-medium">{play.label}</span>
          <NoFillBadge color="text-muted-foreground">{play.type === "ticket" ? "Ticket" : "Doc"}</NoFillBadge>
          {play.type === "ticket" && play.show_when_stage && (
            <NoFillBadge color="text-muted-foreground">{PLAY_STAGE_LABELS[play.show_when_stage]}</NoFillBadge>
          )}
          {!play.enabled && <NoFillBadge color="text-muted-foreground">Disabled</NoFillBadge>}
          {excludedCount > 0 && <NoFillBadge color="text-muted-foreground">{excludedCount} excluded</NoFillBadge>}
        </div>
        {play.description && <p className="text-xs text-muted-foreground">{play.description}</p>}
      </div>
      <div className="flex shrink-0 items-center gap-1">
        {canWrite && (
          <Button variant="ghost" size="sm" aria-label={`Exclude users from ${play.label}`} onClick={() => void openExclusionsDialog()}>
            Exclude users
          </Button>
        )}
        {canWrite && (
          <Button variant="ghost" size="sm" aria-label={`Edit play ${play.label}`} onClick={() => onEdit(play)}>
            Edit
          </Button>
        )}
        {canDelete && (
          <ConfirmDestroyButton
            icon={Trash2}
            idleLabel={`Delete play ${play.label}`}
            disabled={deletePlay.isPending}
            onConfirm={() => deletePlay.mutate(play.id)}
          />
        )}
      </div>
    </li>
  );
};
