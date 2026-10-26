import { WizardAlreadyDeployed } from "@/components/wizard/WizardAlreadyDeployed";
import { WizardBranchesStep } from "@/components/wizard/WizardBranchesStep";
import { WizardDeployStep } from "@/components/wizard/WizardDeployStep";
import { WizardDoneStep } from "@/components/wizard/WizardDoneStep";
import { WizardEnvStep } from "@/components/wizard/WizardEnvStep";
import { WizardPeopleStep } from "@/components/wizard/WizardPeopleStep";
import { WizardProjectCard } from "@/components/wizard/WizardProjectCard";
import { WizardProjectStep } from "@/components/wizard/WizardProjectStep";
import { WizardReachStep } from "@/components/wizard/WizardReachStep";
import { WizardRepositoryStep } from "@/components/wizard/WizardRepositoryStep";
import { WizardScanState } from "@/components/wizard/WizardScanState";
import { WizardServiceStep } from "@/components/wizard/WizardServiceStep";
import { WizardSetupRepositoryStep } from "@/components/wizard/WizardSetupRepositoryStep";
import { WizardStepNeeds } from "@/components/wizard/WizardStepNeeds";
import { WizardStepPanel } from "@/components/wizard/WizardStepPanel";
import { WizardStepSummary } from "@/components/wizard/WizardStepSummary";
import { useWizardBack, useWizardGoTo } from "@/hooks/useWizardNavigation";
import { useWizardEnvKeys, useWizardProject, useWizardStack, useWizardStackId } from "@/hooks/useWizardSetup";
import { useWizardStepActions } from "@/hooks/useWizardStepActions";
import { useWizardDoor, useWizardStepOrder } from "@/hooks/useWizardStepOrder";
import { useWizardStepView } from "@/hooks/useWizardStepView";
import { wizardCanSkip, type WizardStepId } from "@/models/ProjectWizard";
import { useProjectWizardStore } from "@/stores/projectWizardStore";

interface ProjectWizardStepContentProps {
  step: WizardStepId;
}

// Any step opens at any time: missing groundwork is named, and a step that already made something shows it (ADR 0143).
export const ProjectWizardStepContent = ({ step }: ProjectWizardStepContentProps) => {
  const door = useWizardDoor();
  const view = useWizardStepView(step);
  const goTo = useWizardGoTo();
  const onBack = useWizardBack(step);
  const order = useWizardStepOrder();
  const { complete, skip, leaveEarly } = useWizardStepActions(step);
  const projectName = useProjectWizardStore((s) => s.projectName);
  const stackId = useWizardStackId();
  const stack = useWizardStack();
  const project = useWizardProject();
  const envKeys = useWizardEnvKeys();
  const next = (id: WizardStepId): WizardStepId => order[order.indexOf(id) + 1] ?? "done";
  const onSkip = wizardCanSkip(step, door) ? () => void skip() : undefined;
  const runsOn = stack && (
    <>
      {stack.name} runs on <span className="font-mono">{stack.machine}</span>.
    </>
  );

  return (
    <WizardStepPanel step={step}>
      {view.kind === "needs" && <WizardStepNeeds needs={view.needs} onBack={onBack} onSkip={onSkip} />}
      {step === "project" && view.kind === "form" && <WizardProjectStep onDone={(id) => goTo("people", id)} onBack={onBack} />}
      {step === "project" && view.kind === "summary" && (
        <WizardProjectCard name={projectName ?? ""} prefix={project?.prefix ?? ""} onContinue={() => goTo("people")} onBack={onBack} />
      )}
      {step === "people" && view.kind === "form" && project && (
        <WizardPeopleStep project={project} onDone={() => complete(next(step))} onBack={onBack} onSkip={() => void skip()} />
      )}
      {step === "repository" && view.kind === "form" && door === "setup" && (
        <WizardSetupRepositoryStep onDone={() => complete(next(step))} onBack={onBack} onLeaveEarly={leaveEarly} />
      )}
      {step === "repository" && view.kind === "form" && door !== "setup" && (
        <WizardRepositoryStep onDone={() => complete(next(step))} onBack={onBack} />
      )}
      {step === "deploy" && view.kind === "summary" && (
        <WizardStepSummary onContinue={() => complete(next("service"), "service")} onBack={onBack}>
          {runsOn}
        </WizardStepSummary>
      )}
      {step === "deploy" && view.kind === "form" && (
        <WizardDeployStep onNow={() => goTo("service")} onLater={() => void leaveEarly("done")} onBack={onBack} />
      )}
      {step === "service" && view.kind === "summary" && (
        <WizardStepSummary onContinue={() => complete(next(step))} onBack={onBack}>
          {runsOn}
        </WizardStepSummary>
      )}
      {step === "service" && view.kind === "form" && <WizardServiceStep onDone={() => complete(next(step))} onBack={onBack} onSkip={onSkip} />}
      {(step === "deploy" || step === "service") && view.kind === "deployed" && (
        <WizardAlreadyDeployed onContinue={() => goTo("done")} onBack={onBack} />
      )}
      {(step === "deploy" || step === "service") && (view.kind === "scanning" || view.kind === "scanFailed" || view.kind === "nothingFound") && (
        <WizardScanState
          view={view.kind}
          step={step}
          onBack={onBack}
          onLeave={step === "deploy" ? () => void leaveEarly("done") : onSkip}
        />
      )}
      {step === "env" && view.kind === "form" && stackId && (
        <WizardEnvStep stackId={stackId} envKeys={envKeys} onDone={() => complete(next(step))} onBack={onBack} onSkip={onSkip} />
      )}
      {step === "reach" && view.kind === "form" && stackId && (
        <WizardReachStep stackId={stackId} onDone={() => complete(next(step))} onBack={onBack} onSkip={() => void skip()} />
      )}
      {step === "branches" && view.kind === "form" && stackId && (
        <WizardBranchesStep stackId={stackId} onDone={() => complete(next(step))} onBack={onBack} onSkip={() => void skip()} />
      )}
      {step === "done" && view.kind === "form" && <WizardDoneStep onBack={onBack} />}
    </WizardStepPanel>
  );
};
