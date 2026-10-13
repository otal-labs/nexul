import { ProjectsCard } from "@/components/accessgrid/ProjectsCard";
import { SettingsSaveBar } from "@/components/settings/SettingsSaveBar";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { useSaveAccessGrid } from "@/hooks/AccessGridHooks";
import { useAccessDraft } from "@/hooks/useAccessDraft";
import { useFlash } from "@/hooks/useFlash";
import type { AccessGrid } from "@/models/AccessGrid";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface AccessProjectsTabProps {
  grid: AccessGrid;
}

const FORM = "access-projects";

// The Projects grid saved from its own footer, with who finished the setup in its corner.
export const AccessProjectsTab = ({ grid }: AccessProjectsTabProps) => {
  const draft = useAccessDraft(grid);
  const save = useSaveAccessGrid(grid.workspace_id);
  const [saved, flash] = useFlash();
  const { setup } = grid;
  const by = setup.finished_by_login && setup.finished_at && `by ${setup.finished_by_login}, ${formatRelativeTime(setup.finished_at)}`;
  const submit = async () => {
    await save.mutateAsync({ projects: draft.projectChanges });
    draft.clearProjects();
    flash();
  };
  return (
    <form
      id={FORM}
      onSubmit={(event) => {
        event.preventDefault();
        void submit().catch(() => undefined);
      }}
    >
      <ProjectsCard
        grid={grid}
        aside={setup.finished && <SettingsStatus tone="success" detail={by || undefined}>Set up</SettingsStatus>}
        footer={
          <SettingsSaveBar
            form={FORM}
            dirty={draft.projectChanges.length > 0}
            saving={save.isPending}
            saved={saved}
            onDiscard={draft.clearProjects}
            hint="Changes apply to everyone once saved."
          />
        }
      />
    </form>
  );
};
