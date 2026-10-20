import type { ReactNode } from "react";

import { Button } from "@/components/ui/button";
import { WizardFooter } from "@/components/wizard/WizardFooter";

interface WizardStepSummaryProps {
  children: ReactNode;
  onContinue: () => void;
  onBack: () => void;
}

// A step revisited after it made its thing: what it made, and the way on, never a second copy.
export const WizardStepSummary = ({ children, onContinue, onBack }: WizardStepSummaryProps) => (
  <div>
    <div className="text-sm">{children}</div>
    <WizardFooter onBack={onBack}>
      <Button type="button" onClick={onContinue}>
        Continue
      </Button>
    </WizardFooter>
  </div>
);
