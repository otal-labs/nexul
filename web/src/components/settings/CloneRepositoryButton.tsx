import { FolderGit2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { useCloneHarnessProject } from "@/hooks/PairingHooks";

interface CloneRepositoryButtonProps {
  computerId: string;
  repository: string;
  onCloned: (harnessProjectId: string) => void;
}

export const CloneRepositoryButton = ({ computerId, repository, onCloned }: CloneRepositoryButtonProps) => {
  const clone = useCloneHarnessProject(computerId);
  return (
    <Button
      type="button"
      variant="outline"
      size="sm"
      loading={clone.isPending}
      onClick={() => clone.mutate(repository, { onSuccess: (project) => onCloned(project.id) })}
    >
      <FolderGit2 />
      Clone {repository}
    </Button>
  );
};
