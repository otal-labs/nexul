import { useNavigate } from "react-router";

import { ActionButton } from "@/components/action/ActionButton";
import { DangerAction, dangerButtonClass, DangerZone } from "@/components/settings/DangerZone";
import { PeopleLoseProject } from "@/components/settings/PeopleLoseProject";
import { useFetchProjectPeople } from "@/hooks/PeopleHooks";
import { useDeleteProject, useFetchProjectDeleteImpact } from "@/hooks/ProjectHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useEntityActions } from "@/hooks/useEntityActions";
import type { DeleteImpact, Project } from "@/models/Project";

interface ProjectDangerZoneSectionProps {
  project: Project;
}

const plural = (count: number, word: string) => `${count} ${word}${count === 1 ? "" : "s"}`;

const isBlocked = (impact: DeleteImpact) => impact.tickets > 0 || impact.repos > 0 || impact.services > 0;

export const ProjectDangerZoneSection = ({ project }: ProjectDangerZoneSectionProps) => {
  const navigate = useNavigate();
  const deleteProject = useDeleteProject();
  const { data: impact } = useFetchProjectDeleteImpact(project.id);
  const { data: people } = useFetchProjectPeople(project.id);
  const { open: confirmDelete } = useConfirmationDialog();
  // A project that still holds anything can't be removed; the Still holds line under it says what.
  const blocked = !impact || isBlocked(impact);
  const answer = useEntityActions("project", project.id);

  const onDelete = async () => {
    if (blocked) return;
    const losing = people?.length ?? 0;
    const ok = await confirmDelete({
      message: "The project is empty. Removing it can't be undone.",
      title: `Remove ${project.name}?`,
      confirmLabel: "Remove project",
      details: losing > 0 && <PeopleLoseProject count={losing} />,
    });
    if (!ok) return;
    await deleteProject.mutateAsync(project.id);
    navigate("/");
  };

  return (
    <DangerZone title="Can't be undone">
      <DangerAction
        title="Remove project"
        consequence="Removes this project for good. Only an empty project can be removed."
        details={
          impact &&
          isBlocked(impact) && (
            <p className="pt-1 font-mono text-xs text-pretty text-muted-foreground">
              Still holds {plural(impact.tickets, "ticket")} · {plural(impact.repos, "repo")} · {plural(impact.services, "service")}
            </p>
          )
        }
        action={
          <ActionButton
            permission="projects:delete"
            verb="delete projects"
            projectId={project.id}
            answer={answer("projects:delete")}
            variant="outline"
            size="sm"
            liveClassName={blocked ? "" : dangerButtonClass}
            disabled={blocked}
            loading={deleteProject.isPending}
            onClick={() => void onDelete()}
          >
            Remove project
          </ActionButton>
        }
      />
    </DangerZone>
  );
};
