import type { ReactNode } from "react";

import { ShowcaseSurface } from "@/components/showcase/ShowcaseSurface";
import { ProjectWizardRail } from "@/components/wizard/ProjectWizardRail";
import type { WizardStepId } from "@/models/ProjectWizard";

interface ProjectWizardFrameProps {
  step: WizardStepId;
  children: ReactNode;
}

// The step list holds its own column beside the step for the whole wizard, and shrinks to one line when narrow.
export const ProjectWizardFrame = ({ step, children }: ProjectWizardFrameProps) => (
  <ShowcaseSurface quiet>
    <div className="mx-auto w-full max-w-6xl px-6 py-10 lg:px-10 lg:py-16">
      <div className="@container">
        <div className="grid gap-8 @2xl:grid-cols-[13rem_minmax(0,1fr)] @2xl:gap-12 @4xl:grid-cols-[15rem_minmax(0,1fr)] @4xl:gap-16">
          <ProjectWizardRail step={step} />
          <div className="max-w-xl min-w-0 pb-16">{children}</div>
        </div>
      </div>
    </div>
  </ShowcaseSurface>
);
