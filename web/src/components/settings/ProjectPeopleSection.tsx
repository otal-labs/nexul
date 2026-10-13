import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ProjectPeopleAdd } from "@/components/settings/ProjectPeopleAdd";
import { ProjectPeopleGroup } from "@/components/settings/ProjectPeopleGroup";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { useFetchGrants } from "@/hooks/PermissionHooks";
import { usePanelTargets } from "@/hooks/usePanelTargets";
import { projectPeopleRules } from "@/models/ProjectPeople";
import type { Project } from "@/models/Project";

interface ProjectPeopleSectionProps {
  project: Project;
}

// Who's in the project: the roles and people its rules let in, and below them the people kept out (ADR 0148).
export const ProjectPeopleSection = ({ project }: ProjectPeopleSectionProps) => {
  const { data: grants, error, isPending } = useFetchGrants("project", project.id);
  // Rows wait for the names they show, so a role never reads as unknown while the roles load.
  const { ready } = usePanelTargets({ workspaceId: project.workspace_id });
  const rules = grants && ready ? projectPeopleRules(grants) : undefined;

  return (
    <SettingsCard
      id="people"
      title="People in this project"
      description={`Only the roles and people listed here can open ${project.name}. Open one to set what it can do.`}
      footer={grants && <ProjectPeopleAdd project={project} grants={grants} />}
    >
      {(isPending || (grants && !ready)) && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {rules && rules.roles.length + rules.people.length === 0 && <EmptyRow flush>Nobody is in {project.name} yet. Only Owners can open it.</EmptyRow>}
      {rules && (
        <div className="space-y-4">
          {rules.roles.length > 0 && <ProjectPeopleGroup title="Roles" project={project} grants={rules.roles} />}
          {rules.people.length > 0 && <ProjectPeopleGroup title="People" project={project} grants={rules.people} />}
          {rules.keptOut.length > 0 && <ProjectPeopleGroup title="Kept out" project={project} grants={rules.keptOut} keptOut />}
        </div>
      )}
    </SettingsCard>
  );
};
