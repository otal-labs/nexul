import { useShallow } from "zustand/react/shallow";

import { WizardBranchesStep } from "@/components/wizard/WizardBranchesStep";
import { WizardDoneStep } from "@/components/wizard/WizardDoneStep";
import { WizardEnvStep } from "@/components/wizard/WizardEnvStep";
import { WizardProjectStep } from "@/components/wizard/WizardProjectStep";
import { WizardReachStep } from "@/components/wizard/WizardReachStep";
import { WizardRepositoryStep } from "@/components/wizard/WizardRepositoryStep";
import { WizardServiceStep } from "@/components/wizard/WizardServiceStep";
import { WizardStepPanel } from "@/components/wizard/WizardStepPanel";
import { useWizardBack, useWizardGoTo } from "@/hooks/useWizardNavigation";
import { useWizardStepOrder } from "@/hooks/useWizardStepOrder";
import { wizardStepLabel, type WizardStepId } from "@/models/ProjectWizard";
import { useProjectWizardStore } from "@/stores/projectWizardStore";

interface ProjectWizardStepContentProps {
  step: WizardStepId;
}

// Renders only the active step; the URL step is the whole navigation state, so a step's content is just what the
// store has accumulated so far and deep links keep working.
export const ProjectWizardStepContent = ({ step }: ProjectWizardStepContentProps) => {
  const goTo = useWizardGoTo();
  const onBack = useWizardBack(step);
  const order = useWizardStepOrder();
  const { attachStackId, stackId, scanResult } = useProjectWizardStore(
    useShallow((s) => ({ attachStackId: s.attachStackId, stackId: s.stackId, scanResult: s.scanResult })),
  );
  const isAttach = !!attachStackId;
  const showEnv = (scanResult?.env_keys.length ?? 0) > 0;
  const next = order[order.indexOf(step) + 1];
  const continueLabel = next ? `Continue to ${wizardStepLabel(next, isAttach)}` : "Continue";
  // Attach mode skips the Reach rung (the stack already has hostnames), so service and env land on "branches".
  const afterService = (): WizardStepId => {
    if (showEnv) return "env";
    return isAttach ? "branches" : "reach";
  };
  const afterEnv = isAttach ? "branches" : "reach";

  return (
    <WizardStepPanel step={step} title={wizardStepLabel(step, isAttach)}>
      {step === "project" && (
        <WizardProjectStep onDone={() => goTo("repository")} onBack={onBack} continueLabel={continueLabel} />
      )}
      {step === "repository" && <WizardRepositoryStep onDone={() => goTo("service")} />}
      {step === "service" && <WizardServiceStep onDone={() => goTo(afterService())} onBack={onBack} />}
      {step === "env" && <WizardEnvStep onDone={() => goTo(afterEnv)} />}
      {step === "reach" && <WizardReachStep onDone={() => goTo("branches")} onSkip={() => goTo("branches")} />}
      {step === "branches" && <WizardBranchesStep onDone={() => goTo("done")} onSkip={() => goTo("done")} />}
      {step === "done" && stackId && <WizardDoneStep />}
    </WizardStepPanel>
  );
};
