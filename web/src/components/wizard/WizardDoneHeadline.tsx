import { useShallow } from "zustand/react/shallow";

import { useWizardAppRepos, useWizardStack, useWizardStackId } from "@/hooks/useWizardSetup";
import { useProjectWizardStore } from "@/stores/projectWizardStore";

// What setup made: the service deploying, else the repository waiting to deploy, else a project for docs and tickets.
export const WizardDoneHeadline = () => {
  const { projectName, exposureHostname } = useProjectWizardStore(
    useShallow((s) => ({ projectName: s.projectName, exposureHostname: s.exposureHostname })),
  );
  const stackId = useWizardStackId();
  const stack = useWizardStack();
  const repos = useWizardAppRepos();
  const waiting = !stackId && repos;

  return (
    <>
      {stack && (
        <div>
          <p className="text-sm">
            {stack.name} is deploying on {stack.machine}.
          </p>
          {exposureHostname && <p className="mt-1 font-mono text-xs text-muted-foreground">{exposureHostname}</p>}
        </div>
      )}
      {waiting && waiting.length === 0 && <p className="text-sm font-medium">{projectName} is ready for docs and tickets.</p>}
      {waiting && waiting[0] && (
        <p className="text-sm font-medium">
          <span className="font-mono">{waiting[0].full_name}</span> is attached; deploy it whenever it's ready.
        </p>
      )}
    </>
  );
};
