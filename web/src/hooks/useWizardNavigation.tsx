import { useLocation, useNavigate, useSearchParams } from "react-router";

import { useChangeProjectSetup } from "@/hooks/ProjectHooks";
import { usePendingServiceContext } from "@/hooks/useWizardSetup";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useWizardStepOrder } from "@/hooks/useWizardStepOrder";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import type { WizardStepId } from "@/models/ProjectWizard";

// Moves within the wizard keeping the door's query; projectId adds the project Info just made, so a reload keeps it.
export const useWizardGoTo = () => {
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  const [searchParams] = useSearchParams();
  return (step: WizardStepId, projectId?: string) => {
    const params = new URLSearchParams(searchParams);
    if (projectId) params.set("project", projectId);
    const query = params.toString();
    void navigate(wsPath(`/wizard/project/${step}${query ? `?${query}` : ""}`));
  };
};

// Info leaves the wizard; any other step returns to the one before it, which shows what it made instead of making it twice.
export const useWizardBack = (step: WizardStepId): (() => void) => {
  const navigate = useNavigate();
  const location = useLocation();
  const wsPath = useWorkspacePath();
  const goTo = useWizardGoTo();
  const changeSetup = useChangeProjectSetup();
  const pendingContext = usePendingServiceContext();
  const canOpenBoard = useAreaAccess()?.("tickets") ?? false;
  const order = useWizardStepOrder();
  const previous = order[order.indexOf(step) - 1];

  if (previous) return () => goTo(previous);
  return () => {
    const leave = async () => {
      const projectId = useProjectWizardStore.getState().projectId;
      const context = pendingContext();
      if (projectId && context.stack_id) await changeSetup.mutateAsync({ projectId, ...context });
      if (location.key !== "default") return void navigate(-1);
      void navigate(wsPath(canOpenBoard ? "/board" : "/"));
    };
    void leave().catch(() => undefined);
  };
};
