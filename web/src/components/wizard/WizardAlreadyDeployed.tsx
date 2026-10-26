import { Link } from "react-router";

import { WizardStepSummary } from "@/components/wizard/WizardStepSummary";
import { useFetchServices } from "@/hooks/ServiceHooks";
import { useWizardAppRepos } from "@/hooks/useWizardSetup";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { useProjectWizardStore } from "@/stores/projectWizardStore";

interface WizardAlreadyDeployedProps {
  onContinue: () => void;
  onBack: () => void;
}

// Every attached repository already deploys from a service made outside this run: name that service, not a repository to pick.
export const WizardAlreadyDeployed = ({ onContinue, onBack }: WizardAlreadyDeployedProps) => {
  const projectId = useProjectWizardStore((s) => s.projectId);
  const repos = useWizardAppRepos();
  const { data: stacks } = useFetchServices(projectId ?? undefined);
  const wsPath = useWorkspacePath();
  const stack = stacks?.find((s) => repos?.some((r) => s.build_source?.repo_owner === r.owner && s.build_source?.repo_name === r.name));

  return (
    <WizardStepSummary onContinue={onContinue} onBack={onBack}>
      {stack && (
        <>
          <span className="font-mono">
            {stack.build_source?.repo_owner}/{stack.build_source?.repo_name}
          </span>{" "}
          already deploys as{" "}
          <Link to={wsPath(`/stacks/${stack.id}`)} className="text-brand underline-offset-4 hover:underline">
            {stack.name}
          </Link>
          .
        </>
      )}
    </WizardStepSummary>
  );
};
