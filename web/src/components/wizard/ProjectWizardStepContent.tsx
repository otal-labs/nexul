import { useShallow } from "zustand/react/shallow";

import { WizardBranchesStep } from "@/components/wizard/WizardBranchesStep";
import { WizardDoneStep } from "@/components/wizard/WizardDoneStep";
import { WizardEnvStep } from "@/components/wizard/WizardEnvStep";
import { WizardProjectStep } from "@/components/wizard/WizardProjectStep";
import { WizardReachStep } from "@/components/wizard/WizardReachStep";
import { WizardRepositoryStep } from "@/components/wizard/WizardRepositoryStep";
import { WizardServiceStep } from "@/components/wizard/WizardServiceStep";
import { WizardStepNeeds } from "@/components/wizard/WizardStepNeeds";
import { WizardStepPanel } from "@/components/wizard/WizardStepPanel";
import { WizardStepSummary } from "@/components/wizard/WizardStepSummary";
import { useWizardBack, useWizardGoTo } from "@/hooks/useWizardNavigation";
import { useMarkStep, useSetupSession, useWizardEnvKeys, useWizardProject, useWizardStack, useWizardStackId } from "@/hooks/useWizardSetup";
import { useWizardStepOrder } from "@/hooks/useWizardStepOrder";
import { wizardStepLabel, type WizardStepId } from "@/models/ProjectWizard";
import { useProjectWizardStore } from "@/stores/projectWizardStore";

interface ProjectWizardStepContentProps {
  step: WizardStepId;
}

const noService = "There's no service yet.";

// Any step opens at any time: missing groundwork is named, and a step that already made something shows it (ADR 0143).
export const ProjectWizardStepContent = ({ step }: ProjectWizardStepContentProps) => {
  const goTo = useWizardGoTo();
  const onBack = useWizardBack(step);
  const order = useWizardStepOrder();
  const markStep = useMarkStep();
  const { projectId, projectName, attachStackId, candidate } = useProjectWizardStore(
    useShallow((s) => ({
      projectId: s.projectId,
      projectName: s.projectName,
      attachStackId: s.attachStackId,
      candidate: s.candidate,
    })),
  );
  const stackId = useWizardStackId();
  const stack = useWizardStack();
  const project = useWizardProject();
  const session = useSetupSession();
  // The scan lives in this browser; a repository recorded done on another device or before a reload has none to build from.
  const needsRescan = session && project?.setup.steps.repository === "done";
  const isAttach = !!attachStackId;
  const envKeys = useWizardEnvKeys();
  const showEnv = envKeys.length > 0;
  const next = order[order.indexOf(step) + 1];
  const continueLabel = next ? `Continue to ${wizardStepLabel(next, isAttach)}` : "Continue";
  // Attach mode skips the Reach rung (the stack already has hostnames), so service and env land on "branches".
  const afterEnv = isAttach ? "branches" : "reach";
  const afterService = showEnv ? "env" : afterEnv;
  const complete = async (to: WizardStepId) => {
    try {
      await markStep(step, "done");
      goTo(to);
    } catch {
      // The mutation hook shows the error; keep the step open for retry.
    }
  };
  const skip = async () => {
    try {
      await markStep(step, "skipped");
      if (next) goTo(next);
    } catch {
      // The mutation hook shows the error; keep the step open for retry.
    }
  };

  return (
    <WizardStepPanel step={step} title={wizardStepLabel(step, isAttach)}>
      {step === "project" && !projectId && (
        <WizardProjectStep onDone={(id) => goTo("repository", id)} onBack={onBack} continueLabel={continueLabel} />
      )}
      {step === "project" && projectId && (
        <WizardStepSummary continueLabel={continueLabel} onContinue={() => goTo("repository")} onBack={onBack}>
          {projectName} is created. Rename it any time in the project's Settings.
        </WizardStepSummary>
      )}
      {step === "repository" && !projectId && !isAttach && (
        <WizardStepNeeds needs="project" message="A repository belongs to a project. Name the project first." onBack={onBack} />
      )}
      {step === "repository" && (projectId || isAttach) && (
        <WizardRepositoryStep onDone={() => complete("service")} onSkip={isAttach ? undefined : skip} />
      )}
      {step === "service" && stackId && (
        <WizardStepSummary continueLabel={continueLabel} onContinue={() => complete(afterService)} onBack={onBack}>
          {stack && (
            <>
              {stack.name} runs on <span className="font-mono">{stack.machine}</span>.
            </>
          )}
        </WizardStepSummary>
      )}
      {step === "service" && !stackId && !candidate && !needsRescan && (
        <WizardStepNeeds needs="repository" message="A service builds from a repository. Pick one first." onBack={onBack} onSkip={skip} />
      )}
      {step === "service" && !stackId && !candidate && needsRescan && (
        <WizardStepNeeds
          needs="repository"
          message="The repository is recorded, but its scan isn't loaded on this device. Scan it again to set up the service."
          onBack={onBack}
          onSkip={skip}
        />
      )}
      {step === "service" && !stackId && candidate && (
        <WizardServiceStep onDone={() => complete(afterService)} onBack={onBack} onSkip={isAttach ? undefined : skip} />
      )}
      {step === "env" && !stackId && (
        <WizardStepNeeds needs="service" message={`Environment values go to a service. ${noService}`} onBack={onBack} onSkip={skip} />
      )}
      {step === "env" && stackId && (
        <WizardEnvStep stackId={stackId} envKeys={envKeys} onDone={() => complete(afterEnv)} onBack={onBack} onSkip={skip} />
      )}
      {step === "reach" && !stackId && (
        <WizardStepNeeds needs="service" message={`Reach gives a service a hostname. ${noService}`} onBack={onBack} onSkip={skip} />
      )}
      {step === "reach" && stackId && <WizardReachStep stackId={stackId} onDone={() => complete("branches")} onSkip={skip} />}
      {step === "branches" && !stackId && (
        <WizardStepNeeds needs="service" message={`Deploy branches copies a service. ${noService}`} onBack={onBack} onSkip={skip} />
      )}
      {step === "branches" && stackId && <WizardBranchesStep stackId={stackId} onDone={() => complete("done")} onSkip={skip} />}
      {step === "done" && !projectId && !isAttach && (
        <WizardStepNeeds needs="project" message="Finish sets a project up. Name the project first." onBack={onBack} />
      )}
      {step === "done" && (projectId || isAttach) && <WizardDoneStep onBack={onBack} />}
    </WizardStepPanel>
  );
};
