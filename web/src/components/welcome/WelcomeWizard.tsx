import { Navigate, useSearchParams } from "react-router";

import type { WelcomeProgress } from "@nexul/client-core/welcome";

import { WelcomeFrame } from "@/components/welcome/WelcomeFrame";
import { WelcomeFlowContext, useWelcomeFlowState } from "@/hooks/useWelcomeFlow";
import { openingView, type WizardView } from "@/models/WelcomeWizard";

interface OpenWizardProps {
  progress: WelcomeProgress;
  view: WizardView;
}

const OpenWizard = ({ progress, view }: OpenWizardProps) => {
  const flow = useWelcomeFlowState(progress, view);
  return (
    <WelcomeFlowContext value={flow}>
      <WelcomeFrame />
    </WelcomeFlowContext>
  );
};

interface WelcomeWizardProps {
  progress: WelcomeProgress;
}

// Without a step it opens one and writes it into the URL, so a step finishing live never moves the person on by itself.
export const WelcomeWizard = ({ progress }: WelcomeWizardProps) => {
  const [params] = useSearchParams();
  const requested = params.get("step");
  const view = openingView(progress, requested);

  return (
    <>
      {requested !== view && <Navigate replace to={{ search: `?step=${view}` }} />}
      {requested === view && <OpenWizard progress={progress} view={view} />}
    </>
  );
};
