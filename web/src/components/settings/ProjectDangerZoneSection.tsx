import { useNavigate } from "react-router";

import { DangerAction, DangerButton, DangerZone } from "@/components/settings/DangerZone";
import { RestrictedMembersLoseAccess } from "@/components/settings/RestrictedMembersLoseAccess";
import { useDeleteProject, useFetchProjectDeleteImpact } from "@/hooks/ProjectHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import type { DeleteImpact, Project } from "@/models/Project";

interface ProjectDangerZoneSectionProps {
  project: Project;
}

const plural = (count: number, word: string) => `${count} ${word}${count === 1 ? "" : "s"}`;

const isBlocked = (impact: DeleteImpact) => impact.tickets > 0 || impact.repos > 0 || impact.services > 0;

const buildImpactMessage = (impact: DeleteImpact): string =>
  [
    "This project still has work in it:",
    `- ${impact.tickets} ticket${impact.tickets === 1 ? "" : "s"}`,
    `- ${impact.repos} repo${impact.repos === 1 ? "" : "s"}`,
    `- ${impact.services} service${impact.services === 1 ? "" : "s"}`,
    "",
    "Move or delete them first. Every ticket, repo, and service needs a project.",
  ].join("\n");

export const ProjectDangerZoneSection = ({ project }: ProjectDangerZoneSectionProps) => {
  const navigate = useNavigate();
  const deleteProject = useDeleteProject();
  const { data: impact } = useFetchProjectDeleteImpact(project.id);
  const { open: confirmDelete } = useConfirmationDialog();

  const onDelete = async () => {
    if (!impact) return;
    if (isBlocked(impact)) {
      await confirmDelete({
        message: buildImpactMessage(impact),
        title: `Remove ${project.name}?`,
        confirmLabel: "Got it",
        destructive: false,
      });
      return;
    }
    const losing = impact.restricted_members ?? [];
    const ok = await confirmDelete({
      message: "The project is empty. Removing it can't be undone.",
      title: `Remove ${project.name}?`,
      confirmLabel: "Remove project",
      details: losing.length > 0 && <RestrictedMembersLoseAccess members={losing} />,
    });
    if (!ok) return;
    await deleteProject.mutateAsync(project.id);
    navigate("/");
  };

  return (
    <DangerZone>
      <DangerAction
        title="Remove project"
        consequence="Removes this project for good. Only an empty project can be removed."
        details={
          impact &&
          isBlocked(impact) && (
            <p className="pt-1 font-mono text-xs text-muted-foreground">
              Still holds {plural(impact.tickets, "ticket")} · {plural(impact.repos, "repo")} · {plural(impact.services, "service")}
            </p>
          )
        }
        action={
          <DangerButton loading={deleteProject.isPending} onClick={() => void onDelete()}>
            Remove project
          </DangerButton>
        }
      />
    </DangerZone>
  );
};
