import { WizardSteps, type WizardStepId } from "@/models/ProjectWizard";
import { useProjectWizardStore } from "@/stores/projectWizardStore";

// The rungs this run shows: Environment only when the scan found env keys, no Reach when attaching to a stack.
export const useWizardStepOrder = (): WizardStepId[] => {
  const isAttach = useProjectWizardStore((s) => !!s.attachStackId);
  const showEnv = useProjectWizardStore((s) => (s.scanResult?.env_keys.length ?? 0) > 0);
  return WizardSteps.filter((id) => (id !== "env" || showEnv) && (id !== "reach" || !isAttach));
};
