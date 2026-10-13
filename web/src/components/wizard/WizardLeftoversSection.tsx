import { Button } from "@/components/ui/button";
import { useWizardGoTo } from "@/hooks/useWizardNavigation";
import { useWizardAppRepos, useWizardProject, useWizardStackId } from "@/hooks/useWizardSetup";
import { useWizardStepOrder } from "@/hooks/useWizardStepOrder";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import { DeploySteps, wizardStepLabel, type WizardStepId } from "@/models/ProjectWizard";

interface WizardLeftoverRowProps {
  label: string;
  state: string;
  action: string;
  // The button's accessible name, when its words alone don't say what it opens.
  actionName?: string;
  to: WizardStepId;
}

export const WizardLeftoverRow = ({ label, state, action, actionName, to }: WizardLeftoverRowProps) => {
  const goTo = useWizardGoTo();
  return (
    <li className="flex items-center gap-3 py-2">
      <span className="flex-1 text-sm">{label}</span>
      <span className="font-mono text-xs text-muted-foreground">{state}</span>
      <Button type="button" variant="ghost" size="sm" onClick={() => goTo(to)} aria-label={actionName}>
        {action}
      </Button>
    </li>
  );
};

// Finish works with steps left open, each listed with its way back; without a service the deploy path is one row.
export const WizardLeftoversSection = () => {
  const project = useWizardProject();
  const order = useWizardStepOrder();
  const isAttach = useProjectWizardStore((s) => !!s.attachStackId);
  const noService = !useWizardStackId();
  const repos = useWizardAppRepos();
  const folded = (id: WizardStepId) => noService && (id === "repository" || DeploySteps.includes(id));
  const left = order.filter((id) => id !== "done" && project?.setup.steps[id] !== "done" && !folded(id));
  const deploying = noService && repos;
  if (!project || (left.length === 0 && !deploying)) return null;
  return (
    <section aria-labelledby="wizard-leftovers" className="space-y-1">
      <h3 id="wizard-leftovers" className="text-sm font-medium">
        Left for later
      </h3>
      <ul className="divide-y divide-border border-y border-border">
        {left.map((id) => (
          <WizardLeftoverRow
            key={id}
            label={wizardStepLabel(id, isAttach)}
            state={project.setup.steps[id] === "skipped" ? "skipped" : "not done"}
            action="Open"
            actionName={`Open ${wizardStepLabel(id, isAttach)}`}
            to={id}
          />
        ))}
        {deploying && deploying.length === 0 && (
          <WizardLeftoverRow label="Deploying" state="needs a repository" action="Open Repository" to="repository" />
        )}
        {deploying && deploying.length > 0 && <WizardLeftoverRow label="Deploying" state="not started" action="Deploy it" to="service" />}
      </ul>
    </section>
  );
};
