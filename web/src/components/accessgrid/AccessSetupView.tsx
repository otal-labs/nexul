import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { ChannelsCard } from "@/components/accessgrid/ChannelsCard";
import { ProjectsCard } from "@/components/accessgrid/ProjectsCard";
import { ProjectsReadback } from "@/components/accessgrid/ProjectsReadback";
import { PageHeader } from "@/components/PageHeader";
import { useSaveAccessGrid } from "@/hooks/AccessGridHooks";
import { useAccessDraft } from "@/hooks/useAccessDraft";
import { useWorkspaceCrumb } from "@/hooks/useCrumbs";
import { useLeaveGuard } from "@/hooks/useLeaveGuard";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { useAccessDraftStore } from "@/stores/accessDraftStore";
import { countLabel, type AccessGrid } from "@/models/AccessGrid";

interface AccessSetupViewProps {
  grid: AccessGrid;
}

// The one-time setup: both grids on one page and one Finish, which saves them together and ends the setup.
export const AccessSetupView = ({ grid }: AccessSetupViewProps) => {
  const crumb = useWorkspaceCrumb();
  const wsPath = useWorkspacePath();
  const draft = useAccessDraft(grid);
  const save = useSaveAccessGrid(grid.workspace_id);
  useLeaveGuard(draft.projectChanges.length + draft.channelChanges.length > 0, wsPath("/access"), useAccessDraftStore.getState().clear);

  const finish = async () => {
    await save.mutateAsync({ projects: draft.projectChanges, channels: draft.channelChanges, finish: true });
    useAccessDraftStore.getState().clear();
    toast.success("Access is set", { description: "People see their projects now. Change it any time on the Access page." });
  };
  const facts = `${countLabel(grid.projects.length, "project", "projects")} · ${countLabel(grid.roles.length, "role", "roles")}`;

  return (
    <>
      <div className="mb-8">
        <PageHeader crumbs={[crumb]} title="Who's in each project" meta={<span className="font-mono text-xs text-muted-foreground">{facts}</span>} />
        <p className="mt-5 max-w-2xl text-sm text-pretty text-muted-foreground">
          This update rebuilt permissions, so every project starts with nobody in it and only Owners can see them. Choose who's in each
          project; you can change it any time on the Access page.
        </p>
      </div>
      <div className="space-y-6">
        <ProjectsCard grid={grid} footer={<ProjectsReadback grid={grid} />} />
        <ChannelsCard
          grid={grid}
          alignWithProjects
          footer={
            <>
              <span className="text-sm text-muted-foreground">Leave every box ticked to keep channels as they are.</span>
              <Button size="sm" loading={save.isPending} onClick={() => void finish().catch(() => undefined)}>
                Finish setup
              </Button>
            </>
          }
        />
      </div>
    </>
  );
};
