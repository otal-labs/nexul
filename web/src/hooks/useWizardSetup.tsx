import { useEffect } from "react";
import { useShallow } from "zustand/react/shallow";

import { useChangeProjectSetup, useFetchProject } from "@/hooks/ProjectHooks";
import { useFetchStacks } from "@/hooks/StackHooks";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import { inSetup, type SetupMark } from "@/models/Project";
import type { WizardStepId } from "@/models/ProjectWizard";
import type { Stack } from "@/models/Stack";

// The project the wizard is working on, as the server records it; undefined until Info has made one.
export const useWizardProject = () => {
  const projectId = useProjectWizardStore((s) => s.projectId);
  return useFetchProject(projectId ?? undefined).data;
};

// Records what happened at a step on the project; a failed write toasts and the wizard still moves on.
export const useMarkStep = () => {
  const projectId = useProjectWizardStore((s) => s.projectId);
  const change = useChangeProjectSetup();
  return (step: WizardStepId, mark: SetupMark) => {
    if (projectId && step !== "done") change.mutate({ projectId, steps: { [step]: mark } });
  };
};

// Setup continued on another device: the service it made there is the one Reach and Deploy branches work on here.
export const useSeedSetupStack = () => {
  const project = useWizardProject();
  const { stackId, attachStackId, setStackId, setName, setMachine } = useProjectWizardStore(
    useShallow((s) => ({
      stackId: s.stackId,
      attachStackId: s.attachStackId,
      setStackId: s.setStackId,
      setName: s.setName,
      setMachine: s.setMachine,
    })),
  );
  const wanted = !!project && inSetup(project) && project.setup.steps.service === "done" && !stackId && !attachStackId;
  const { data: stacks } = useFetchStacks(project?.id, wanted);
  const newest = stacks
    ?.filter((s) => !s.derived_from)
    .reduce<Stack | undefined>((best, s) => (!best || s.created_at > best.created_at ? s : best), undefined);

  useEffect(() => {
    if (!wanted || !newest) return;
    setStackId(newest.id);
    setName(newest.name);
    setMachine(newest.machine);
  }, [wanted, newest, setStackId, setName, setMachine]);
};
