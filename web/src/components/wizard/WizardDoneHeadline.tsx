import { useShallow } from "zustand/react/shallow";

import { useWizardStack } from "@/hooks/useWizardSetup";
import { useProjectWizardStore } from "@/stores/projectWizardStore";

// The service setup ends with: deploying when this run made it, else running; a run without one says nothing here.
export const WizardDoneHeadline = () => {
  const { madeId, exposureHostname } = useProjectWizardStore(
    useShallow((s) => ({ madeId: s.stackId, exposureHostname: s.exposureHostname })),
  );
  const stack = useWizardStack();

  if (!stack) return null;
  return (
    <div>
      <p className="text-sm">
        {stack.name} {madeId ? "is deploying on" : "runs on"} {stack.machine}.
      </p>
      {exposureHostname && <p className="mt-1 font-mono text-xs text-muted-foreground">{exposureHostname}</p>}
    </div>
  );
};
