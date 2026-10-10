import { useEffect } from "react";
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

// Navigation waits for the project to remember the step; a failed write leaves it open for retry.
export const useMarkStep = () => {
  const projectId = useProjectWizardStore((s) => s.projectId);
  const change = useChangeProjectSetup();
  const project = useWizardProject();
  return (step: WizardStepId, mark: SetupMark) => {
    if (!projectId || step === "done") return;
    const { stackId, scanResult } = useProjectWizardStore.getState();
    return change.mutateAsync({
      projectId,
      steps: { [step]: mark },
      ...(step === "service" && mark === "done" && stackId && {
        stack_id: stackId,
        env_keys: scanResult?.env_keys ?? project?.setup.env_keys ?? [],
      }),
    });
  };
};

// Setup continued on another device: the service it made there is the one Reach and Deploy branches work on here.
export const useSeedSetupStack = () => {
  const project = useWizardProject();
  const [searchParams] = useSearchParams();
  const { stackId, attachStackId, setStackId, setName, setMachine } = useProjectWizardStore(
    useShallow((s) => ({
      stackId: s.stackId,
      attachStackId: s.attachStackId,
      setStackId: s.setStackId,
      setName: s.setName,
      setMachine: s.setMachine,
    })),
  );
  const wanted =
    !!project &&
    (inSetup(project) || searchParams.has("revisit")) &&
    !!project.setup.stack_id &&
    !stackId &&
    !attachStackId;
  const { data: stack } = useFetchStack(wanted ? project.setup.stack_id : undefined);

  useEffect(() => {
    if (!wanted || !stack || stack.project_id !== project?.id) return;
    setStackId(stack.id);
    setName(stack.name);
    setMachine(stack.machine);
  }, [wanted, stack, project?.id, setStackId, setName, setMachine]);
};

export const useWizardEnvKeys = (): string[] => {
  const project = useWizardProject();
  const scan = useProjectWizardStore((s) => s.scanResult);
  const stackId = useProjectWizardStore((s) => s.stackId);
  if (scan) return scan.env_keys;
  if (!project || (!inSetup(project) && stackId !== project.setup.stack_id)) return [];
  if (stackId && stackId !== project.setup.stack_id) return [];
  return project.setup.env_keys ?? [];
};
