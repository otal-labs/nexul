import { useState, type ReactNode } from "react";

import { useWizardStepOrder } from "@/hooks/useWizardStepOrder";
import { wizardStepDescription, type WizardStepId } from "@/models/ProjectWizard";
import { cn } from "@/lib/utils";

interface WizardStepPanelProps {
  step: WizardStepId;
  title: string;
  children: ReactNode;
}

// Slides the step in from the side it was reached from, so moving on and going back read as opposite directions.
export const WizardStepPanel = ({ step, title, children }: WizardStepPanelProps) => {
  const index = useWizardStepOrder().indexOf(step);
  const [seen, setSeen] = useState(index);
  const [forward, setForward] = useState(true);
  if (index !== seen) {
    setForward(index > seen);
    setSeen(index);
  }
  const description = wizardStepDescription(step);

  return (
    <div
      key={step}
      className={cn(
        "animate-in fade-in-0 duration-180 ease-out motion-reduce:animate-none",
        forward ? "slide-in-from-right-2" : "slide-in-from-left-2",
      )}
    >
      <h2 className="text-base font-semibold tracking-tight">{title}</h2>
      {description && <p className="mt-1 text-sm text-muted-foreground">{description}</p>}
      <div className="mt-4">{children}</div>
    </div>
  );
};
