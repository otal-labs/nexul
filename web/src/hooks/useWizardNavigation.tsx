import { useLocation, useNavigate, useSearchParams } from "react-router";

import { useChangeProjectSetup } from "@/hooks/ProjectHooks";
import { usePendingServiceContext, useWizardProject, useWizardStackId } from "@/hooks/useWizardSetup";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useWizardDoor, useWizardStepOrder } from "@/hooks/useWizardStepOrder";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { boardPath } from "@/models/Project";
import { railRows, type WizardStepId } from "@/models/ProjectWizard";

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

// Leaving saves the service the run was making, then goes back where the wizard was opened from or to the project's board.
export const useWizardLeave = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const wsPath = useWorkspacePath();
  const changeSetup = useChangeProjectSetup();
  const pendingContext = usePendingServiceContext();
  const canOpenBoard = useAreaAccess()?.("tickets") ?? false;
  const project = useWizardProject();

  return (back: boolean) => {
    const leave = async () => {
      const projectId = useProjectWizardStore.getState().projectId;
      const context = pendingContext();
      if (projectId && context.stack_id) await changeSetup.mutateAsync({ projectId, ...context });
      if (back && location.key !== "default") return void navigate(-1);
      if (!canOpenBoard) return void navigate(wsPath("/"));
      void navigate(wsPath(project ? boardPath(project) : "/board"));
    };
    void leave().catch(() => undefined);
  };
};

// The first step leaves the wizard; any other returns to the row above it on the rail, which shows what it made instead of making it twice.
export const useWizardBack = (step: WizardStepId): (() => void) => {
  const goTo = useWizardGoTo();
  const leave = useWizardLeave();
  const door = useWizardDoor();
  const rows = railRows(door, useWizardStepOrder(), step, !!useWizardStackId());
  const previous = rows[rows.findIndex((row) => row.id === step) - 1]?.id;

  if (previous) return () => goTo(previous);
  return () => leave(true);
};
