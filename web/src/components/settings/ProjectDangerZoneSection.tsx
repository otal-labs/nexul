import { TriangleAlert } from "lucide-react";
import { useNavigate } from "react-router";

import { SettingsCard } from "@/components/settings/SettingsCard";
import { Button } from "@/components/ui/button";
import { useDeleteProject, useFetchProjectDeleteImpact } from "@/hooks/ProjectHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import type { DeleteImpact, Project } from "@/models/Project";

interface ProjectDangerZoneSectionProps {
  project: Project;
}

const buildImpactMessage = (impact: DeleteImpact): string =>
  [
    "This project still has affected work:",
    `- ${impact.tickets} ticket${impact.tickets === 1 ? "" : "s"}`,
    `- ${impact.repos} repo${impact.repos === 1 ? "" : "s"}`,
    `- ${impact.services} service${impact.services === 1 ? "" : "s"}`,
    "",
    "Move or delete them first — every ticket, repo, and service must belong to a project.",
  ].join("\n");

// SettingsCard's `danger` prop supplies the red-outlined border and icon-badge tone here.
export const ProjectDangerZoneSection = ({ project }: ProjectDangerZoneSectionProps) => {
  const navigate = useNavigate();
  const deleteProject = useDeleteProject();
  const { data: impact } = useFetchProjectDeleteImpact(project.id);
  const { open: confirmDelete } = useConfirmationDialog();

  const onDelete = async () => {
    if (!impact) return;
    const blocked = impact.tickets > 0 || impact.repos > 0 || impact.services > 0;
    if (blocked) {
      await confirmDelete({
        message: buildImpactMessage(impact),
        title: `Remove ${project.name}?`,
        confirmLabel: "Got it",
        destructive: false,
      });
      return;
    }
    const ok = await confirmDelete({
      message: "This project is empty and can be removed. This cannot be undone.",
      title: `Remove ${project.name}?`,
      confirmLabel: "Remove",
    });
    if (!ok) return;
    await deleteProject.mutateAsync(project.id);
    navigate("/");
  };

  return (
    <SettingsCard id="danger-zone" title="Danger zone" danger icon={TriangleAlert}>
      <div className="flex flex-col items-start justify-between gap-4 sm:flex-row sm:items-center">
        <p className="text-sm text-muted-foreground">
          Remove this project and all associated data. This cannot be undone.
        </p>
        <Button variant="destructive" className="shrink-0" onClick={() => void onDelete()}>
          Remove project
        </Button>
      </div>
    </SettingsCard>
  );
};
