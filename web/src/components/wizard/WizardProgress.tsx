import { WizardProgressItem, type WizardProgressState } from "@/components/wizard/WizardProgressItem";
import { useWizardGoTo } from "@/hooks/useWizardNavigation";
import { useWizardProject } from "@/hooks/useWizardSetup";
import { useWizardStepOrder } from "@/hooks/useWizardStepOrder";
import { wizardStepLabel, type WizardStepId } from "@/models/ProjectWizard";
import type { Project } from "@/models/Project";
import { useProjectWizardStore } from "@/stores/projectWizardStore";

interface WizardProgressProps {
  step: WizardStepId;
}

// What the project records for a step; Done is done once setup was finished.
const stateOf = (id: WizardStepId, step: WizardStepId, project: Project | undefined): WizardProgressState => {
  if (id === step) return "current";
  if (id === "done") return project?.setup.finished ? "done" : "unvisited";
  return project?.setup.steps[id] ?? "unvisited";
};

// The horizontal row above the step. Labels show from a 42rem container up; narrower, one line under the row names
// the current step instead, because six labels would clip.
export const WizardProgress = ({ step }: WizardProgressProps) => {
  const order = useWizardStepOrder();
  const isAttach = useProjectWizardStore((s) => !!s.attachStackId);
  const project = useWizardProject();
  const goTo = useWizardGoTo();
  const current = order.indexOf(step);

  return (
    <div className="@container">
      <ol aria-label="Project wizard steps" className="flex">
        {order.map((id, index) => (
          <WizardProgressItem
            key={id}
            label={wizardStepLabel(id, isAttach)}
            state={stateOf(id, step, project)}
            first={index === 0}
            reached={index <= current}
            onSelect={() => goTo(id)}
          />
        ))}
      </ol>
      <p className="mt-3 text-center font-mono text-xs @2xl:hidden">
        <span className="font-bold text-foreground">{wizardStepLabel(step, isAttach)}</span>{" "}
        <span className="text-muted-foreground">
          {current + 1} / {order.length}
        </span>
      </p>
    </div>
  );
};
