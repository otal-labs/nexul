import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { BranchRulesForm } from "@/components/wizard/BranchRulesForm";
import { WizardFooter } from "@/components/wizard/WizardFooter";
import { WizardSkipLink } from "@/components/wizard/WizardSkipLink";
import { useFetchExposures } from "@/hooks/DnsHooks";
import { useFetchMachineNetworks, useFetchStack, useFetchStackServices } from "@/hooks/StackHooks";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import { defaultNetwork } from "@/models/Stack";

interface WizardBranchesStepProps {
  stackId: string;
  onDone: () => void;
  onBack: () => void;
  onSkip: () => void;
}

export const WizardBranchesStep = ({ stackId, onDone, onBack, onSkip }: WizardBranchesStepProps) => {
  const candidate = useProjectWizardStore((s) => s.candidate);
  const stack = useFetchStack(stackId);
  const services = useFetchStackServices(stackId);
  const exposures = useFetchExposures();
  const networks = useFetchMachineNetworks(stack.data?.machine ?? "", stack.data ? defaultNetwork(stack.data) : "");

  const isPending = stack.isPending || services.isPending || exposures.isPending || networks.isPending;
  const error = stack.error ?? services.error ?? exposures.error ?? networks.error;
  const exposure = exposures.data?.find((e) => services.data?.some((c) => c.id === e.service_id));
  const port = exposure?.port ?? candidate?.reachable?.port;

  const ready = stack.data && services.data && exposures.data && networks.data;

  return (
    <>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} title="Couldn't load what this machine runs." />}
      {stack.data && services.data && exposures.data && networks.data && (
        <BranchRulesForm
          stack={stack.data}
          exposure={exposure}
          defaultPort={port ? String(port) : ""}
          networks={networks.data}
          onDone={onDone}
          onBack={onBack}
          onSkip={onSkip}
        />
      )}
      {!ready && <WizardFooter onBack={onBack} skip={<WizardSkipLink onClick={onSkip} />} />}
    </>
  );
};
