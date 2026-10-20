import { useWizardGoTo } from "@/hooks/useWizardNavigation";
import { useMarkStep, useWizardStackId } from "@/hooks/useWizardSetup";
import { useWizardDoor, useWizardStepOrder } from "@/hooks/useWizardStepOrder";
import type { SetupMark } from "@/models/Project";
import { isDeployStep, wizardSkipPlan, type WizardStepId } from "@/models/ProjectWizard";

// What leaving a step records, then where it goes; a failed write keeps the step open, its hook already toasted.
export const useWizardStepActions = (step: WizardStepId) => {
  const goTo = useWizardGoTo();
  const order = useWizardStepOrder();
  const markStep = useMarkStep();
  const door = useWizardDoor();
  const stack = !!useWizardStackId();

  // Deploy's summary records the service it passes over, since the server has no step for Deploy itself.
  const complete = async (to: WizardStepId, as: WizardStepId = step) => {
    try {
      await markStep(as, "done");
      goTo(to);
    } catch {
      return;
    }
  };
  const skip = async () => {
    const { to, alsoSkipped } = wizardSkipPlan(step, door, order, stack);
    try {
      await markStep(step, "skipped", Object.fromEntries(alsoSkipped.map((id) => [id, "skipped" as const])));
      if (to) goTo(to);
    } catch {
      return;
    }
  };
  // A project set up without a service records every deploy step skipped, so Continue setup lands on Done.
  const leaveEarly = async (repository: SetupMark) => {
    const deploySkipped = Object.fromEntries(order.filter(isDeployStep).map((id) => [id, "skipped" as const]));
    try {
      await markStep("repository", repository, deploySkipped);
      goTo("done");
    } catch {
      return;
    }
  };

  return { complete, skip, leaveEarly };
};
