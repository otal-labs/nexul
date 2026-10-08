import { PlusIcon } from "lucide-react";

import { AddRepoForm } from "@/components/project/AddRepoForm";
import { RepoRow } from "@/components/project/RepoRow";
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
    <div className="mt-3">
      <div className="flex items-center justify-between gap-2">
        <span className="text-sm font-medium">Repositories</span>
        <Button variant="ghost" size="sm" onClick={() => void onAdd()}>
          <PlusIcon className="size-3.5" />
          Add repo
        </Button>
      </div>
      {repos && repos.length === 0 && (
        <p className="mt-2 text-sm text-muted-foreground">No repositories associated yet.</p>
      )}
      {repos && repos.length > 0 && (
        <div className="mt-2">
          <div className="flex items-center gap-2 border-b border-border px-2 pb-1.5 text-xs text-muted-foreground">
            <span className="flex-1">Repository</span>
            <span className="w-20 shrink-0">Provider</span>
            <span className="w-8 shrink-0" aria-hidden />
          </div>
          <ul className="divide-y divide-border">
            {repos.map((repo) => (
              <RepoRow key={`${repo.owner}/${repo.name}`} repo={repo} />
            ))}
          </ul>
        </div>
      )}
    </div>
  );
};
