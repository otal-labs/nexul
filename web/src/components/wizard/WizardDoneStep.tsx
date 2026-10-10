import { useNavigate } from "react-router";
import { useShallow } from "zustand/react/shallow";

import { Button } from "@/components/ui/button";
import { WizardFooter } from "@/components/wizard/WizardFooter";
import { WizardInterviewOffer } from "@/components/wizard/WizardInterviewOffer";
import { WizardLeftoversSection } from "@/components/wizard/WizardLeftoversSection";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useInterviewOffer } from "@/hooks/useInterviewOffer";
import { useChangeProjectSetup, useFetchProjects } from "@/hooks/ProjectHooks";
import { usePendingServiceContext, useSetupSession, useWizardStack, useWizardStackId } from "@/hooks/useWizardSetup";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import { interviewPath, projectTokenById } from "@/models/Project";

interface WizardDoneStepProps {
  onBack: () => void;
}

// The last step: what the wizard built, what was left for later, the interview offer, and Finish, which sets the project up.
export const WizardDoneStep = ({ onBack }: WizardDoneStepProps) => {
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  const can = useAreaAccess();
  const { name, exposureHostname, projectId, projectName } = useProjectWizardStore(
    useShallow((s) => ({
      name: s.name,
      exposureHostname: s.exposureHostname,
      projectId: s.projectId,
      projectName: s.projectName,
    })),
  );
  const stackId = useWizardStackId();
  const stack = useWizardStack();
  const session = useSetupSession();
  const { data: projects } = useFetchProjects();
  const offer = useInterviewOffer(projectId, projectName ?? name);
  const changeSetup = useChangeProjectSetup();
  const pendingContext = usePendingServiceContext();
  const token = projectId ? projectTokenById(projects ?? [], projectId) : "";

  const go = (to: string) => {
    navigate(wsPath(to));
  };
  // Leaving any other way than the interview counts as skipping it, so it asks first.
  const saveAndGo = async (to: string) => {
    try {
      const context = pendingContext();
      if (projectId && context.stack_id) await changeSetup.mutateAsync({ projectId, ...context });
      go(to);
    } catch {
      return;
    }
  };
  const leave = async (to: string) => {
    if (await offer.confirmSkip()) await saveAndGo(to);
  };
  const finish = async () => {
    if (!projectId || !(await offer.confirmSkip())) return;
    try {
      if (session) await changeSetup.mutateAsync({ projectId, ...pendingContext(), finished: true });
      go(can?.("tickets") ? `/board/${token}` : "/");
    } catch {
      // The hook toasts the failure; the step stays so Finish can be pressed again.
    }
  };

  return (
    <div className="space-y-5">
      {stack && (
        <div>
          <p className="text-sm">
            {stack.name} is deploying on {stack.machine}.
          </p>
          {exposureHostname && <p className="mt-1 font-mono text-xs text-muted-foreground">{exposureHostname}</p>}
        </div>
      )}
      <WizardLeftoversSection />
      {offer.pending && projectId && (
        <WizardInterviewOffer
          projectName={projectName ?? name}
          onStart={() => void saveAndGo(interviewPath(token))}
          onSkip={() => void offer.confirmSkip()}
        />
      )}
      {offer.skipped && (
        <p className="text-sm text-muted-foreground">Interview skipped. A banner on the project reminds you until it exists.</p>
      )}
      <WizardFooter onBack={onBack}>
        {stackId && can?.("stacks") && (
          <Button variant="outline" onClick={() => void leave(`/stacks/${stackId}`)}>
            View stack
          </Button>
        )}
        {stackId && can?.("topology") && (
          <Button variant="outline" onClick={() => void leave("/topology")}>
            View on the canvas
          </Button>
        )}
        <Button variant={offer.pending ? "outline" : "default"} onClick={() => void finish()} loading={changeSetup.isPending}>
          Finish
        </Button>
      </WizardFooter>
    </div>
  );
};
