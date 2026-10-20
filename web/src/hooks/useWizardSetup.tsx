import { useSearchParams } from "react-router";
import { useShallow } from "zustand/react/shallow";

import { useChangeProjectSetup, useFetchProject, useFetchProjectRepos } from "@/hooks/ProjectHooks";
import { useFetchStack } from "@/hooks/StackHooks";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import { inSetup, type SetupMark } from "@/models/Project";
import type { ServerStepId, WizardStepId } from "@/models/ProjectWizard";
import { RepoRole } from "@/enums/Project";

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

// Deploy and Done are the browser's own: the server's setup never hears of them.
const isServerStep = (step: WizardStepId): step is ServerStepId => step !== "deploy" && step !== "done";

// Navigation waits for the project to remember the step; a failed write leaves it open for retry.
export const useMarkStep = () => {
  const projectId = useProjectWizardStore((s) => s.projectId);
  const session = useSetupSession();
  const change = useChangeProjectSetup();
  const pendingContext = usePendingServiceContext();
  return (step: WizardStepId, mark: SetupMark, also: Partial<Record<ServerStepId, SetupMark>> = {}) => {
    if (!projectId || !session || !isServerStep(step)) return;
    const context = pendingContext();
    return change.mutateAsync({
      projectId,
      ...context,
      steps: { ...context.steps, ...also, [step]: mark },
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

// The project's repositories a service could build from; undefined until they load.
export const useWizardAppRepos = () => {
  const projectId = useProjectWizardStore((s) => s.projectId);
  return useFetchProjectRepos(projectId ?? undefined).data?.filter((r) => r.role !== RepoRole.Tests);
};

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

// The tests question as this run answered it, else as the project saved it, so a revisit opens on the saved answer.
export const useWizardTestsAnswer = () => {
  const project = useWizardProject();
  const { projectId, testsLocation, testsRepo } = useProjectWizardStore(
    useShallow((s) => ({ projectId: s.projectId, testsLocation: s.testsLocation, testsRepo: s.testsRepo })),
  );
  const attached = useFetchProjectRepos(projectId ?? undefined).data?.find((r) => r.role === RepoRole.Tests);
  const savedRepo = attached && { owner: attached.owner, name: attached.name, full_name: attached.full_name, provider: attached.connector_id };
  return { testsLocation: testsLocation ?? (project?.tests_location || null), testsRepo: testsRepo ?? savedRepo ?? null };
};
