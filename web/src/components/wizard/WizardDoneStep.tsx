import { useNavigate } from "react-router";
import { useShallow } from "zustand/react/shallow";

import { Button } from "@/components/ui/button";
import { WizardDoneHeadline } from "@/components/wizard/WizardDoneHeadline";
import { WizardFooter } from "@/components/wizard/WizardFooter";
import { WizardInterviewOffer } from "@/components/wizard/WizardInterviewOffer";
import { WizardLeftoversSection } from "@/components/wizard/WizardLeftoversSection";
import { WizardNextPlacesSection } from "@/components/wizard/WizardNextPlacesSection";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useInterviewOffer } from "@/hooks/useInterviewOffer";
import { useChangeProjectSetup, useFetchProjects } from "@/hooks/ProjectHooks";
import { usePendingServiceContext, useSetupSession, useWizardStackId } from "@/hooks/useWizardSetup";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { interviewPath, projectTokenById } from "@/models/Project";

interface WizardDoneStepProps {
  onBack: () => void;
}

// The last step: what the wizard built, what was left for later, the interview offer, and Finish, which sets the project up.
export const WizardDoneStep = ({ onBack }: WizardDoneStepProps) => {
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  const can = useAreaAccess();
  const { name, projectId, projectName } = useProjectWizardStore(
    useShallow((s) => ({
      name: s.name,
      projectId: s.projectId,
      projectName: s.projectName,
    })),
  );
  const selectProject = useWorkspaceStore((s) => s.selectProject);
  const stackId = useWizardStackId();
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
  // Lands in the project it set up, wherever the path goes.
  const finish = async (path: string) => {
    if (!projectId || !(await offer.confirmSkip())) return;
    try {
      if (session) await changeSetup.mutateAsync({ projectId, ...pendingContext(), finished: true });
      selectProject(projectId);
      navigate(path);
    } catch {
      // The hook toasts the failure; the step stays so Finish can be pressed again.
    }
  };

  return (
    <div className="space-y-5">
      <WizardDoneHeadline />
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
      {!stackId && <WizardNextPlacesSection onGo={(path) => void finish(path)} />}
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
        <Button
          variant={offer.pending ? "outline" : "default"}
          onClick={() => void finish(wsPath(can?.("tickets") ? `/board/${token}` : "/"))}
          loading={changeSetup.isPending}
        >
          Finish
        </Button>
      </WizardFooter>
    </div>
  );
};
