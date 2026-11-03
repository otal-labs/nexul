import { FolderGit2 } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { useInstallTool } from "@/hooks/ComputerToolHooks";
import { useCloneHarnessProject } from "@/hooks/PairingHooks";
import { missingToolOf } from "@/models/ComputerTools";
import { errorMessage } from "@/api/client";

interface CloneRepositoryButtonProps {
  computerId: string;
  repository: string;
  disabled?: boolean;
  onCloned: (harnessProjectId: string) => void;
}

export const CloneRepositoryButton = ({ computerId, repository, disabled, onCloned }: CloneRepositoryButtonProps) => {
  const clone = useCloneHarnessProject(computerId);
  const { install } = useInstallTool(computerId);
  // A clone that failed for want of git says so and offers to install it there; the clone hook toasts every other failure.
  const offerInstall = (error: unknown) => {
    const missing = missingToolOf(error);
    if (!missing) return;
    toast.error(errorMessage(error), { action: { label: `Install ${missing.tool}`, onClick: () => void install(missing.tool) } });
  };
  return (
    <Button
      type="button"
      variant="outline"
      size="sm"
      disabled={disabled}
      loading={clone.isPending}
      onClick={() => clone.mutate(repository, { onSuccess: (project) => onCloned(project.id), onError: offerInstall })}
    >
      <FolderGit2 />
      Clone {repository}
    </Button>
  );
};
