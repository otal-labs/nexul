import { useSearchParams } from "react-router";
import { useShallow } from "zustand/react/shallow";

import { useChangeProjectSetup, useFetchProject } from "@/hooks/ProjectHooks";
import { useFetchStack } from "@/hooks/StackHooks";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import { inSetup, type SetupMark } from "@/models/Project";
import type { WizardStepId } from "@/models/ProjectWizard";

// The project the wizard is working on, as the server records it; undefined until Info has made one.
export const useWizardProject = () => {
  const projectId = useProjectWizardStore((s) => s.projectId);
  return useFetchProject(projectId ?? undefined).data;
};

// The recorded setup is this run's to read and write while the project is in setup, or when Settings reopened it to
// revisit; the Add a service and Attach doors work on a new service and leave setup alone (ADR 0143).
export const useSetupSession = (): boolean => {
  const project = useWizardProject();
  const [searchParams] = useSearchParams();
  if (!project) return false;
  if (searchParams.has("revisit")) return true;
  return inSetup(project) && !searchParams.has("add") && !searchParams.has("stack");
};

// Navigation waits for the project to remember the step; a failed write leaves it open for retry.
export const useMarkStep = () => {
  const projectId = useProjectWizardStore((s) => s.projectId);
  const session = useSetupSession();
  const change = useChangeProjectSetup();
  const pendingContext = usePendingServiceContext();
  return (step: WizardStepId, mark: SetupMark) => {
    if (!projectId || !session || step === "done") return;
    const context = pendingContext();
    return change.mutateAsync({
      projectId,
      ...context,
      steps: { ...context.steps, [step]: mark },
    });
  };
};

// The service the wizard works on: the one this run made, else the one setup recorded (it may have been made on
// another device), once its stack loads and proves to be the project's own.
export const useWizardStackId = (): string | null => {
  const project = useWizardProject();
  const session = useSetupSession();
  const madeId = useProjectWizardStore((s) => s.stackId);
  const { data: recorded } = useFetchStack(session && !madeId ? project?.setup.stack_id : undefined);
  if (madeId) return madeId;
  return recorded && recorded.project_id === project?.id ? recorded.id : null;
};

export const useWizardStack = () => useFetchStack(useWizardStackId() ?? undefined).data;

// A scan describes one build source: once a service exists, its detected keys stay with that service.
export const useWizardEnvKeys = (): string[] => {
  const project = useWizardProject();
  const session = useSetupSession();
  const { madeId, scanResult, serviceEnvKeys } = useProjectWizardStore(
    useShallow((s) => ({ madeId: s.stackId, scanResult: s.scanResult, serviceEnvKeys: s.serviceEnvKeys })),
  );
  if (madeId) return serviceEnvKeys;
  if (session && project?.setup.stack_id) return project.setup.env_keys ?? [];
  return scanResult?.env_keys ?? [];
};

// What the run made that setup has not recorded yet; empty when it made nothing, or when the run leaves setup alone.
export const usePendingServiceContext = () => {
  const project = useWizardProject();
  const session = useSetupSession();
  return () => {
    const { stackId, serviceEnvKeys } = useProjectWizardStore.getState();
    const recordedKeys = project?.setup.env_keys ?? [];
    const recorded = stackId === project?.setup.stack_id && serviceEnvKeys.join("\n") === recordedKeys.join("\n");
    if (!session || !stackId || recorded) return {};
    return { stack_id: stackId, env_keys: serviceEnvKeys, steps: { service: "done" as const } };
  };
};
