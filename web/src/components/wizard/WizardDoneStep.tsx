import { useNavigate } from "react-router";
import { useShallow } from "zustand/react/shallow";

import { Button } from "@/components/ui/button";
import { WizardInterviewOffer } from "@/components/wizard/WizardInterviewOffer";
import { useInterviewOffer } from "@/hooks/useInterviewOffer";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import { interviewPath, projectTokenById } from "@/models/Project";

// Terminal rung: what the wizard built, the interview offer while the project has none, and links to the stack and canvas.
export const WizardDoneStep = () => {
  const navigate = useNavigate();
  const { name, machine, exposureHostname, stackId, projectId, projectName } = useProjectWizardStore(
    useShallow((s) => ({
      name: s.name,
      machine: s.machine,
      exposureHostname: s.exposureHostname,
      stackId: s.stackId,
      projectId: s.projectId,
      projectName: s.projectName,
    })),
  );
  const reset = useProjectWizardStore((s) => s.reset);
  const { data: projects } = useFetchProjects();
  const offer = useInterviewOffer(projectId, projectName ?? name);

  const go = (to: string) => {
    reset();
    navigate(to);
  };
  // Leaving any other way than the interview counts as skipping it, so it asks first.
  const leave = async (to: string) => {
    if (await offer.confirmSkip()) go(to);
  };

  return (
    <div className="space-y-5">
      <div>
        <p className="text-sm">
          {name} is deploying on {machine}.
        </p>
        {exposureHostname && (
          <p className="mt-1 font-mono text-xs text-muted-foreground">{exposureHostname}</p>
        )}
      </div>
      {offer.pending && projectId && (
        <WizardInterviewOffer
          projectName={projectName ?? name}
          onStart={() => go(interviewPath(projectTokenById(projects ?? [], projectId)))}
          onSkip={() => void offer.confirmSkip()}
        />
      )}
      {offer.skipped && (
        <p className="text-sm text-muted-foreground">
          Interview skipped. A banner on the project reminds you until it exists.
        </p>
      )}
      <div className="flex flex-wrap gap-3">
        {stackId && (
          <Button variant="outline" onClick={() => void leave(`/stacks/${stackId}`)}>
            View stack
          </Button>
        )}
        <Button variant={offer.pending ? "outline" : "default"} onClick={() => void leave("/topology")}>
          View on the canvas
        </Button>
      </div>
    </div>
  );
};
