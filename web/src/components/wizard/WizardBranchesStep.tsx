import { useShallow } from "zustand/react/shallow";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { BranchRulesForm } from "@/components/wizard/BranchRulesForm";
import { WizardSkipLink } from "@/components/wizard/WizardSkipLink";
import { useFetchExposures } from "@/hooks/DnsHooks";
import { useFetchMachineNetworks, useFetchStack, useFetchStackServices } from "@/hooks/StackHooks";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import { defaultNetwork } from "@/models/Stack";

interface WizardBranchesStepProps {
  onDone: () => void;
  onSkip: () => void;
}

export const WizardBranchesStep = ({ onDone, onSkip }: WizardBranchesStepProps) => {
  const { stackId, candidate } = useProjectWizardStore(
    useShallow((s) => ({ stackId: s.stackId, candidate: s.candidate })),
  );
  const stack = useFetchStack(stackId ?? undefined);
  const services = useFetchStackServices(stackId ?? undefined);
  const exposures = useFetchExposures();
  const networks = useFetchMachineNetworks(stack.data?.machine ?? "", stack.data ? defaultNetwork(stack.data) : "");

  const isPending = stack.isPending || services.isPending || exposures.isPending || networks.isPending;
  const error = stack.error ?? services.error ?? exposures.error ?? networks.error;
  const exposure = exposures.data?.find((e) => services.data?.some((c) => c.id === e.service_id));
  const port = exposure?.port ?? candidate?.reachable?.port;

  return (
    <>
      {isPending && <LoadingDisplay />}
      {error && (
        <div className="space-y-4">
          <ErrorDisplay error={error} title="Couldn't load what this machine runs" />
          <WizardSkipLink onClick={onSkip} />
        </div>
      )}
      {stack.data && services.data && exposures.data && networks.data && (
        <BranchRulesForm
          stack={stack.data}
          exposure={exposure}
          defaultPort={port ? String(port) : ""}
          networks={networks.data}
          onDone={onDone}
          onSkip={onSkip}
        />
      )}
    </>
  );
};
