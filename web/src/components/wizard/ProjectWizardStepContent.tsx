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
import { useMarkStep } from "@/hooks/useWizardSetup";
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
  const { projectId, projectName, attachStackId, stackId, name, machine, candidate, scanResult } = useProjectWizardStore(
    useShallow((s) => ({
      projectId: s.projectId,
      projectName: s.projectName,
      attachStackId: s.attachStackId,
      stackId: s.stackId,
      name: s.name,
      machine: s.machine,
      candidate: s.candidate,
      scanResult: s.scanResult,
    })),
  );
  const isAttach = !!attachStackId;
  const showEnv = (scanResult?.env_keys.length ?? 0) > 0;
  const next = order[order.indexOf(step) + 1];
  const continueLabel = next ? `Continue to ${wizardStepLabel(next, isAttach)}` : "Continue";
  // Attach mode skips the Reach rung (the stack already has hostnames), so service and env land on "branches".
  const afterEnv = isAttach ? "branches" : "reach";
  const afterService = showEnv ? "env" : afterEnv;
  const complete = (to: WizardStepId) => {
    markStep(step, "done");
    goTo(to);
  };
  const skip = () => {
    markStep(step, "skipped");
    if (next) goTo(next);
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
        <WizardStepSummary continueLabel={continueLabel} onContinue={() => goTo(afterService)} onBack={onBack}>
          {name} runs on <span className="font-mono">{machine}</span>.
        </WizardStepSummary>
      )}
      {step === "service" && !stackId && !candidate && (
        <WizardStepNeeds needs="repository" message="A service builds from a repository. Pick one first." onBack={onBack} onSkip={skip} />
      )}
      {step === "service" && !stackId && candidate && (
        <WizardServiceStep onDone={() => complete(afterService)} onBack={onBack} onSkip={isAttach ? undefined : skip} />
      )}
      {step === "env" && !stackId && (
        <WizardStepNeeds needs="service" message={`Environment values go to a service. ${noService}`} onBack={onBack} />
      )}
      {step === "env" && stackId && <WizardEnvStep onDone={() => complete(afterEnv)} />}
      {step === "reach" && !stackId && (
        <WizardStepNeeds needs="service" message={`Reach gives a service a hostname. ${noService}`} onBack={onBack} onSkip={skip} />
      )}
      {step === "reach" && stackId && <WizardReachStep onDone={() => complete("branches")} onSkip={skip} />}
      {step === "branches" && !stackId && (
        <WizardStepNeeds needs="service" message={`Deploy branches copies a service. ${noService}`} onBack={onBack} onSkip={skip} />
      )}
      {step === "branches" && stackId && <WizardBranchesStep onDone={() => complete("done")} onSkip={skip} />}
      {step === "done" && !projectId && !isAttach && (
        <WizardStepNeeds needs="project" message="Finish sets a project up. Name the project first." onBack={onBack} />
      )}
      {step === "done" && (projectId || isAttach) && <WizardDoneStep onBack={onBack} />}
    </WizardStepPanel>
  );
};
