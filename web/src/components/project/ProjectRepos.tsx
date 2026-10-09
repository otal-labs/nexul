import { PlusIcon } from "lucide-react";
import { AnimatePresence } from "motion/react";

import { EnterList } from "@/components/EnterList";
import { AddRepoForm } from "@/components/project/AddRepoForm";
import { RepoRow } from "@/components/project/RepoRow";
import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { Button } from "@/components/ui/button";
import { useFetchProjectRepos } from "@/hooks/ProjectHooks";
import { useFormDialog } from "@/hooks/useFormDialog";
import { AddProjectRepoFormSchema, type AddProjectRepoFormData } from "@/models/Project";

interface ProjectReposProps {
  projectId: string;
}

export const ProjectRepos = ({ projectId }: ProjectReposProps) => {
  const { data: repos, isPending, error } = useFetchProjectRepos(projectId);
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
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {repos && repos.length === 0 && <EmptyRow>No repositories yet. Add one to link its pull requests to tickets.</EmptyRow>}
      {repos && repos.length > 0 && (
        <EnterList className="relative divide-y divide-border overflow-hidden rounded-md border border-border">
          <AnimatePresence initial={false} mode="popLayout">
            {repos.map((repo, index) => (
              <RepoRow key={`${repo.owner}/${repo.name}`} index={index} repo={repo} />
            ))}
          </AnimatePresence>
        </EnterList>
      )}
    </SettingsCard>
  );
};
