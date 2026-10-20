import { useShallow } from "zustand/react/shallow";

import { useScanTarget } from "@/hooks/useWaitingRepos";
import { useWizardStackId } from "@/hooks/useWizardSetup";
import { useWizardDoor } from "@/hooks/useWizardStepOrder";
import { wizardStepView, type StepView, type WizardFacts, type WizardStepId } from "@/models/ProjectWizard";
import { useProjectWizardStore } from "@/stores/projectWizardStore";

// The facts a step's view turns on, read from the run, the project and the repository's scan.
export const useWizardFacts = (): WizardFacts => {
  const door = useWizardDoor();
  const { projectId, candidate, scanned, seed } = useProjectWizardStore(
    useShallow((s) => ({ projectId: s.projectId, candidate: !!s.candidate, scanned: !!s.scanResult, seed: s.seed.status })),
  );
  const stack = !!useWizardStackId();
  const { target, pending } = useScanTarget();
  const scan = seed === "idle" ? (scanned && !candidate ? "empty" : "idle") : seed;
  return { door, project: !!projectId, stack, appRepo: !!target || pending, candidate, scan };
};

export const useWizardStepView = (step: WizardStepId): StepView => wizardStepView(step, useWizardFacts());
