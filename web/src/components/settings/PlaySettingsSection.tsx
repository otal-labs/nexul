import { PlusIcon } from "lucide-react";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { PlayForm } from "@/components/play/PlayForm";
import { PlayRow } from "@/components/play/PlayRow";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { Button } from "@/components/ui/button";
import { useFormDialog } from "@/hooks/useFormDialog";
import { useFetchWorkspacePlays } from "@/hooks/PlayHooks";
import { SavePlayFormSchema, type Play, type SavePlayFormData } from "@/models/Play";
import { useWorkspaceStore } from "@/stores/workspaceStore";

interface PlaySettingsSectionProps {
  canWrite: boolean;
  canDelete: boolean;
}

const emptyDefaults = (): SavePlayFormData => ({
  label: "",
  type: "ticket",
  description: "",
  instructions: "",
  enabled: true,
  show_when_stage: "progress",
  excluded_project_ids: [],
});

const defaultsFrom = (play: Play): SavePlayFormData => ({
  label: play.label,
  type: play.type,
  description: play.description,
  instructions: play.instructions,
  enabled: play.enabled,
  show_when_stage: play.show_when_stage ?? "",
  excluded_project_ids: play.excluded_project_ids,
});

// Gated on plays:read/write/delete by the settings page, same gate-in-parent pattern as RoleSettingsSection.
export const PlaySettingsSection = ({ canWrite, canDelete }: PlaySettingsSectionProps) => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: plays, isPending, error } = useFetchWorkspacePlays(workspaceId);
  const { open } = useFormDialog();

  const openDialog = (play: Play | null) =>
    open<SavePlayFormData>({
      title: play ? `Edit ${play.label}` : "New play",
      schema: SavePlayFormSchema,
      okLabel: play ? "Save" : "Create play",
      form: <PlayForm workspaceId={workspaceId} {...(play ? { editing: play } : {})} />,
      formOptions: { defaultValues: play ? defaultsFrom(play) : emptyDefaults() },
    });

  return (
    <SettingsCard
      id="plays"
      title="Plays"
      description="Pre-configured Agent turns members can fire from a ticket, a doc, or a project's Interview page.
        Seeded with three defaults; editable and deletable like any other play."
      footer={
        canWrite && (
          <Button type="button" onClick={() => void openDialog(null)}>
            <PlusIcon className="size-4" />
            New play
          </Button>
        )
      }
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {plays && plays.length === 0 && <NoDataDisplay message="No plays yet" />}
      {plays && plays.length > 0 && (
        <ul className="divide-y divide-border overflow-hidden rounded-md border">
          {plays.map((play) => (
            <PlayRow
              key={play.id}
              play={play}
              workspaceId={workspaceId}
              canWrite={canWrite}
              canDelete={canDelete}
              onEdit={() => void openDialog(play)}
            />
          ))}
        </ul>
      )}
    </SettingsCard>
  );
};
