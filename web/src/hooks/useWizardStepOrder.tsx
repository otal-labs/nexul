import { useParams } from "react-router";

import { useWizardEnvKeys } from "@/hooks/useWizardSetup";
import { WizardSteps, type WizardStepId } from "@/models/ProjectWizard";
import { useProjectWizardStore } from "@/stores/projectWizardStore";

export const useWizardStepOrder = (): WizardStepId[] => {
  const isAttach = useProjectWizardStore((s) => !!s.attachStackId);
  const { step } = useParams();
  const showEnv = useWizardEnvKeys().length > 0 || step === "env";
  return WizardSteps.filter((id) => (id !== "env" || showEnv) && (id !== "reach" || !isAttach));
};
