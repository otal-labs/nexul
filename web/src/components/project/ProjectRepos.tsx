import { PlusIcon } from "lucide-react";

import { EnterList } from "@/components/EnterList";
import { AddRepoForm } from "@/components/project/AddRepoForm";
import { RepoRow } from "@/components/project/RepoRow";
import { EmptyRow } from "@/components/EmptyRow";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { Button } from "@/components/ui/button";
import { useFetchProjectRepos } from "@/hooks/ProjectHooks";
import { useFormDialog } from "@/hooks/useFormDialog";
import { AddProjectRepoFormSchema, type AddProjectRepoFormData } from "@/models/Project";

interface ProjectReposProps {
  projectId: string;
}

export const ProjectRepos = ({ projectId }: ProjectReposProps) => {
  const { data: repos } = useFetchProjectRepos(projectId);
  const { open: openAdd } = useFormDialog();

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
      title="Repositories"
      description="Where this project's code lives. A tests repository is never deployed."
      footer={
        <Button variant="outline" size="sm" onClick={() => void onAdd()}>
          <PlusIcon className="size-4" />
          Add repo
        </Button>
      }
    >
      {repos && repos.length === 0 && <EmptyRow>No repositories yet. Add one to link its pull requests to tickets.</EmptyRow>}
      {repos && repos.length > 0 && (
        <EnterList className="divide-y divide-border">
          {repos.map((repo) => (
            <RepoRow key={`${repo.owner}/${repo.name}`} repo={repo} />
          ))}
        </EnterList>
      )}
    </SettingsCard>
  );
};
