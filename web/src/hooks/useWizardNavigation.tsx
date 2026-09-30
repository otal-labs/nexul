import { useLocation, useNavigate, useSearchParams } from "react-router";

import { useAreaAccess } from "@/hooks/AccessHooks";
import { useWizardStepOrder } from "@/hooks/useWizardStepOrder";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import type { WizardStepId } from "@/models/ProjectWizard";
import { useProjectWizardStore } from "@/stores/projectWizardStore";

// Moves within the wizard, keeping the door's query (?project=, ?stack=) so a step still knows how it was entered.
export const useWizardGoTo = () => {
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  const [searchParams] = useSearchParams();
  return (step: WizardStepId) => {
    const query = searchParams.toString();
    void navigate(wsPath(`/wizard/project/${step}${query ? `?${query}` : ""}`));
  };
};

// Info created the project and a stack exists after Service, so going back into either would create a second one.
export const useCanRevisitStep = () => {
  const stackId = useProjectWizardStore((s) => s.stackId);
  return (id: WizardStepId) => id !== "project" && !stackId;
};

// Info leaves the wizard (nothing exists yet); any other rung returns to the one before it when that is safe to revisit.
export const useWizardBack = (step: WizardStepId): (() => void) | undefined => {
  const navigate = useNavigate();
  const location = useLocation();
  const wsPath = useWorkspacePath();
  const goTo = useWizardGoTo();
  const canOpenBoard = useAreaAccess()?.("tickets") ?? false;
  const canRevisit = useCanRevisitStep();
  const order = useWizardStepOrder();
  const previous = order[order.indexOf(step) - 1];

  const leave = () => {
    // "default" is the first entry of the session, so there is no in-app page to return to.
    if (location.key !== "default") return void navigate(-1);
    void navigate(wsPath(canOpenBoard ? "/board" : "/"));
  };

  if (step === "project") return leave;
  if (previous && canRevisit(previous)) return () => goTo(previous);
  return undefined;
};
