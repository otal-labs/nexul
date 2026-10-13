import type { ReactNode } from "react";

import { Button } from "@/components/ui/button";
import { WizardFooter } from "@/components/wizard/WizardFooter";
import { useWelcomeFlow } from "@/hooks/useWelcomeFlow";

interface WelcomeFooterProps {
  // Hidden once the step is done: there is nothing left to skip.
  skip: boolean;
  // The one primary; a step that has none moves on only through Skip.
  children?: ReactNode;
}

// The project wizard's footer: Back on the left, then a ghost Skip for now, then the primary.
export const WelcomeFooter = ({ skip, children }: WelcomeFooterProps) => {
  const { steps, view, back, skip: skipStep, skipping } = useWelcomeFlow();
  return (
    <WizardFooter
      onBack={view === steps[0] ? undefined : back}
      skip={
        skip && (
          <Button type="button" variant="ghost" className="text-muted-foreground" loading={skipping} onClick={skipStep}>
            Skip for now
          </Button>
        )
      }
    >
      {children}
    </WizardFooter>
  );
};

// A step done already moves on with Continue alone.
export const ContinueButton = ({ disabled = false }: { disabled?: boolean }) => {
  const { next } = useWelcomeFlow();
  return (
    <Button type="button" disabled={disabled} onClick={next}>
      Continue
    </Button>
  );
};
