import { useShallow } from "zustand/react/shallow";

import { useScanTarget, useWaitingReposState } from "@/hooks/useWaitingRepos";
import { useWizardAppRepos, useWizardStackId } from "@/hooks/useWizardSetup";
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
  const { waiting } = useWaitingReposState(door === "setup" ? (projectId ?? undefined) : undefined);
  const appRepos = useWizardAppRepos();
  const deployed = !!waiting && waiting.length === 0 && !!appRepos && appRepos.length > 0;
  const scan = seed === "idle" ? (scanned && !candidate ? "empty" : "idle") : seed;
  return { door, project: !!projectId, stack, appRepo: !!target || pending, deployed, candidate, scan };
};

export const useWizardStepView = (step: WizardStepId): StepView => wizardStepView(step, useWizardFacts());
