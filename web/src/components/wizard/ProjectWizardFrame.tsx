import type { ReactNode } from "react";

import { ShowcaseSurface } from "@/components/showcase/ShowcaseSurface";
import { Button } from "@/components/ui/button";
import { ProjectWizardRail } from "@/components/wizard/ProjectWizardRail";
import { WizardShell } from "@/components/wizard/WizardShell";
import { useWizardLeave } from "@/hooks/useWizardNavigation";
import { useWizardProject } from "@/hooks/useWizardSetup";
import { useWizardDoor } from "@/hooks/useWizardStepOrder";
import { inSetup } from "@/models/Project";
import type { WizardStepId } from "@/models/ProjectWizard";

interface ProjectWizardFrameProps {
  step: WizardStepId;
  children: ReactNode;
}

export const ProjectWizardFrame = ({ step, children }: ProjectWizardFrameProps) => {
  const leave = useWizardLeave();
  const door = useWizardDoor();
  const project = useWizardProject();
  // A project in setup resumes from the sidebar's Continue setup; the other doors and a project not yet made don't.
  const resumable = door === "setup" && !!project && inSetup(project);

  return (
    <ShowcaseSurface quiet>
      <WizardShell
        rail={<ProjectWizardRail step={step} />}
        exit={
          <Button variant="ghost" size="sm" className="-mr-2 h-7 px-2 text-muted-foreground" onClick={() => leave(false)}>
            {resumable ? "Finish later" : "Exit"}
          </Button>
        }
      >
        {children}
      </WizardShell>
    </ShowcaseSurface>
  );
};
