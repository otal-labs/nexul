import { CloneRepositoryButton } from "@/components/settings/CloneRepositoryButton";
import { PairAgainToClone } from "@/components/settings/PairAgainToClone";
import { useCanCloneOnComputer } from "@/hooks/PairingHooks";

interface CloneRepositoriesSectionProps {
  computerId: string;
  // owner/name of each repository the computer holds no T3 project for.
  repositories: string[];
  onCloned: (harnessProjectId: string) => void;
}

export const CloneRepositoriesSection = ({ computerId, repositories, onCloned }: CloneRepositoriesSectionProps) => {
  const { data: canClone } = useCanCloneOnComputer(computerId);

  return (
    <div className="space-y-2">
      <p className="text-xs text-muted-foreground">
        Not on this computer yet. Cloning puts it beside the computer's other T3 projects.
      </p>
      {canClone === false && <PairAgainToClone computerId={computerId} />}
      <div className="flex flex-wrap gap-2">
        {repositories.map((repository) => (
          <CloneRepositoryButton key={repository} computerId={computerId} repository={repository} disabled={canClone === false} onCloned={onCloned} />
        ))}
      </div>
    </div>
  );
};
