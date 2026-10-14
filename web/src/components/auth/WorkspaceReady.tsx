import { WizardConfirmation } from "@/components/auth/WizardConfirmation";
import { ShowcaseSurface } from "@/components/showcase/ShowcaseSurface";

// The owner wizard's last beat, before the Welcome wizard takes over with the owner's own steps.
export const WorkspaceReady = () => (
  <ShowcaseSurface quiet>
    <div className="flex min-h-full items-center justify-center px-6">
      <WizardConfirmation title="Workspace ready" subtitle="Next, what's yours to set up: GitHub and your computer." />
    </div>
  </ShowcaseSurface>
);
