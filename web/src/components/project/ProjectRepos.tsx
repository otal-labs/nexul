import { PlusIcon } from "lucide-react";
import { EnterList } from "@/components/EnterList";
import { AddRepoForm } from "@/components/project/AddRepoForm";
import { RepoRow } from "@/components/project/RepoRow";
import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { settingsRowListClass } from "@/components/settings/SettingsRow";
import { ActionButton } from "@/components/action/ActionButton";
import { useRowGlide } from "@/hooks/useRowGlide";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useFetchProjectRepos } from "@/hooks/ProjectHooks";
import { useFetchServices } from "@/hooks/ServiceHooks";
import { useFormDialog } from "@/hooks/useFormDialog";
import { AddProjectRepoFormSchema, type AddProjectRepoFormData } from "@/models/Project";

interface ProjectReposProps {
  projectId: string;
}

export const ProjectRepos = ({ projectId }: ProjectReposProps) => {
  const { data: repos, isPending, error } = useFetchProjectRepos(projectId);
  const { open: openAdd } = useFormDialog();
  const { ref: glideRef, prepare: prepareGlide } = useRowGlide();
  const canReadStacks = useAreaAccess(projectId)?.("stacks") ?? false;
  const { data: services } = useFetchServices(projectId, canReadStacks);
  const buildsFrom = (repo: { owner: string; name: string }) =>
    services?.filter((s) => s.build_source?.repo_owner === repo.owner && s.build_source?.repo_name === repo.name).length;

  const onAdd = async () => {
    await openAdd<AddProjectRepoFormData>({
      title: "Add repository",
      schema: AddProjectRepoFormSchema,
      okLabel: "Add repository",
      form: <AddRepoForm projectId={projectId} />,
      formOptions: { defaultValues: { owner: "", name: "", connectorId: "github" } },
    });
  };

  return (
    <SettingsCard
      id="repositories"
      title="Attached"
      description="Where this project's code lives. A tests repository is never deployed."
      footer={
        <ActionButton permission="projects:write" projectId={projectId} verb="add repositories" variant="outline" size="sm" icon={PlusIcon} onClick={() => void onAdd()}>
          Add repo
        </ActionButton>
      }
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {repos && repos.length === 0 && <EmptyRow>No repositories yet. Add one to link its pull requests to tickets.</EmptyRow>}
      {repos && repos.length > 0 && (
        <div ref={glideRef}>
          <EnterList className={settingsRowListClass}>
            {repos.map((repo) => (
              <RepoRow key={`${repo.owner}/${repo.name}`} projectId={projectId} repo={repo} services={buildsFrom(repo)} onLeave={prepareGlide} />
            ))}
          </EnterList>
        </div>
      )}
    </SettingsCard>
  );
};
