import { Button } from "@/components/ui/button";
import { useWizardGoTo } from "@/hooks/useWizardNavigation";
import { useWizardProject } from "@/hooks/useWizardSetup";
import { useWizardStepOrder } from "@/hooks/useWizardStepOrder";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import { wizardStepLabel, type WizardStepId } from "@/models/ProjectWizard";

interface WizardLeftoverRowProps {
  step: WizardStepId;
  skipped: boolean;
}

export const WizardLeftoverRow = ({ step, skipped }: WizardLeftoverRowProps) => {
  const goTo = useWizardGoTo();
  const label = wizardStepLabel(step, useProjectWizardStore((s) => !!s.attachStackId));
  return (
    <li className="flex items-center gap-3 py-2">
      <span className="flex-1 text-sm">{label}</span>
      <span className="font-mono text-xs text-muted-foreground">{skipped ? "skipped" : "not done"}</span>
      <Button type="button" variant="ghost" size="sm" onClick={() => goTo(step)} aria-label={`Open ${label}`}>
        Open
      </Button>
    </li>
  );
};

// Finish works with steps left open; this names each one with the way back to it.
export const WizardLeftoversSection = () => {
  const project = useWizardProject();
  const order = useWizardStepOrder();
  const left = order.filter((id) => id !== "done" && project?.setup.steps[id] !== "done");
  if (!project || left.length === 0) return null;
  return (
    <section aria-labelledby="wizard-leftovers" className="space-y-1">
      <h3 id="wizard-leftovers" className="text-sm font-medium">
        Left for later
      </h3>
      <ul className="divide-y divide-border border-y border-border">
        {left.map((id) => (
          <WizardLeftoverRow key={id} step={id} skipped={project.setup.steps[id] === "skipped"} />
        ))}
      </ul>
    </section>
  );
};
