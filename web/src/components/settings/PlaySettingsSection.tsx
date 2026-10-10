import { PlusIcon } from "lucide-react";
import { EnterList } from "@/components/EnterList";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { EmptyRow } from "@/components/EmptyRow";
import { PlayDialogBody } from "@/components/play/PlayDialogBody";
import { PlayDialogHeader } from "@/components/play/PlayDialogHeader";
import { PlayForm } from "@/components/play/PlayForm";
import { PlayRow } from "@/components/play/PlayRow";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { Button } from "@/components/ui/button";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useRowGlide } from "@/hooks/useRowGlide";
import { useFormDialog } from "@/hooks/useFormDialog";
import { useFetchWorkspacePlays } from "@/hooks/PlayHooks";
import { SavePlayFormSchema, type Play, type SavePlayFormData } from "@/models/Play";
import { usePlayDialogStore } from "@/stores/playDialogStore";
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
  const { ref: glideRef, prepare: prepareGlide } = useRowGlide();
  const can = useAreaAccess();
  const setTab = usePlayDialogStore((s) => s.setTab);

  const openEdit = (play: Play) => {
    const title = `Edit ${play.label}`;
    // An interview play takes no auto plays, so its dialog stays the plain form.
    const tabbed = play.type !== "interview" && (can?.("autoPlays") ?? false);
    setTab("play");
    return open<SavePlayFormData>({
      title,
      schema: SavePlayFormSchema,
      okLabel: "Save",
      ...(tabbed ? { header: <PlayDialogHeader title={title} />, dialogClassName: "sm:max-w-[36rem]" } : {}),
      form: tabbed ? <PlayDialogBody workspaceId={workspaceId} play={play} /> : <PlayForm workspaceId={workspaceId} editing={play} />,
      formOptions: { defaultValues: defaultsFrom(play) },
    });
  };

  const openCreate = () =>
    open<SavePlayFormData>({
      title: "New play",
      schema: SavePlayFormSchema,
      okLabel: "Create play",
      form: <PlayForm workspaceId={workspaceId} />,
      formOptions: { defaultValues: emptyDefaults() },
    });

  return (
    <SettingsCard
      id="plays"
      title="Plays"
      description="Agent turns members start with one button on a ticket, a doc, or an Interview page. Every workspace starts with a set you can edit or delete."
      footer={
        canWrite && (
          <Button type="button" onClick={() => void openCreate()}>
            <PlusIcon className="size-4" />
            New play
          </Button>
        )
      }
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {plays && plays.length === 0 && <EmptyRow>No plays yet</EmptyRow>}
      {plays && plays.length > 0 && (
        <div ref={glideRef}>
          <EnterList className="divide-y divide-border overflow-hidden rounded-md border">
            {plays.map((play) => (
              <PlayRow
                key={play.id}
                play={play}
                workspaceId={workspaceId}
                canWrite={canWrite}
                canDelete={canDelete}
                onEdit={() => void openEdit(play)}
                onLeave={prepareGlide}
              />
            ))}
          </EnterList>
        </div>
      )}
    </SettingsCard>
  );
};
