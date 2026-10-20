import type { ReactNode } from "react";

import { Logo } from "@/components/Logo";
import { ShowcaseSurface } from "@/components/showcase/ShowcaseSurface";
import { Button } from "@/components/ui/button";
import { ProjectWizardRail } from "@/components/wizard/ProjectWizardRail";
import { useWizardLeave } from "@/hooks/useWizardNavigation";
import { useWizardProject } from "@/hooks/useWizardSetup";
import { useWizardDoor } from "@/hooks/useWizardStepOrder";
import { inSetup } from "@/models/Project";
import type { WizardStepId } from "@/models/ProjectWizard";

interface ProjectWizardFrameProps {
  step: WizardStepId;
  children: ReactNode;
}

// The step list holds its own column beside the step for the whole wizard, and shrinks to one line when narrow.
export const ProjectWizardFrame = ({ step, children }: ProjectWizardFrameProps) => {
  const leave = useWizardLeave();
  const door = useWizardDoor();
  const project = useWizardProject();
  // A project in setup resumes from the sidebar's Continue setup; the other doors and a project not yet made don't.
  const resumable = door === "setup" && !!project && inSetup(project);

  return (
    <ShowcaseSurface quiet>
      <div className="mx-auto w-full max-w-6xl px-6 py-6 lg:px-10">
        <header className="flex items-center justify-between">
          <Logo className="size-7" />
          <Button variant="ghost" size="sm" className="text-muted-foreground" onClick={() => leave(false)}>
            {resumable ? "Finish later" : "Exit"}
          </Button>
        </header>
        <div className="@container mt-10 lg:mt-16">
          <div className="grid gap-8 @2xl:grid-cols-[13rem_minmax(0,1fr)] @2xl:gap-12 @4xl:grid-cols-[15rem_minmax(0,1fr)] @4xl:gap-16">
            <ProjectWizardRail step={step} />
            <div className="max-w-xl min-w-0 pb-16">{children}</div>
          </div>
        </div>
      </div>
    </ShowcaseSurface>
  );
};
