import { EnterList } from "@/components/EnterList";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { EmptyRow } from "@/components/EmptyRow";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ProjectPeopleAccessRow } from "@/components/settings/ProjectPeopleAccessRow";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { useFetchProjectAccess } from "@/hooks/ProjectHooks";
import type { Project } from "@/models/Project";

interface ProjectPeopleAccessSectionProps {
  project: Project;
}

// Read-only on purpose: access is set on the person, in Team, so there is one place to change it.
export const ProjectPeopleAccessSection = ({ project }: ProjectPeopleAccessSectionProps) => {
  const { data: access, error, isPending } = useFetchProjectAccess(project.id);

  return (
    <SettingsCard
      id="people-with-access"
      title="People with access"
      description={`Restricted members who can open ${project.name}. Everyone else reaches it through their role.`}
      footer={<p className="text-xs text-muted-foreground">Change access from Team.</p>}
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {access && access.length === 0 && <EmptyRow>No restricted member can open {project.name}.</EmptyRow>}
      {access && access.length > 0 && (
        <EnterList className="-my-2 divide-y divide-border">
          {access.map((entry) => (
            <ProjectPeopleAccessRow key={entry.user_id} entry={entry} />
          ))}
        </EnterList>
      )}
    </SettingsCard>
  );
};
