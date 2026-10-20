import { useState, type ReactNode } from "react";

import { displayTitleClass } from "@/components/PageHeader";
import { useWizardDoor, useWizardStepOrder } from "@/hooks/useWizardStepOrder";
import { wizardStepLead, wizardStepTitle, type WizardStepId } from "@/models/ProjectWizard";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import { cn } from "@/lib/utils";

interface WizardStepPanelProps {
  step: WizardStepId;
  children: ReactNode;
}

// The step's own question as the page title, then its body, sliding in from the side it was reached from so moving on and going back read as opposite directions.
export const WizardStepPanel = ({ step, children }: WizardStepPanelProps) => {
  const door = useWizardDoor();
  const projectName = useProjectWizardStore((s) => s.projectName);
  const index = useWizardStepOrder().indexOf(step);
  const [seen, setSeen] = useState(index);
  const [forward, setForward] = useState(true);
  if (index !== seen) {
    setForward(index > seen);
    setSeen(index);
  }
  const lead = wizardStepLead(step);

  return (
    <div
      key={step}
      className={cn(
        "animate-in fade-in-0 duration-180 ease-out motion-reduce:animate-none",
        forward ? "slide-in-from-right-2" : "slide-in-from-left-2",
      )}
    >
      <h1 className={cn(displayTitleClass, "text-[2rem]")}>{wizardStepTitle(step, door, projectName)}</h1>
      {lead && <p className="mt-2 text-muted-foreground">{lead}</p>}
      <div className="mt-8">{children}</div>
    </div>
  );
};
