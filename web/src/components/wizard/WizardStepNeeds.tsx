import { Button } from "@/components/ui/button";
import { WizardFooter } from "@/components/wizard/WizardFooter";
import { WizardSkipLink } from "@/components/wizard/WizardSkipLink";
import { useWizardGoTo } from "@/hooks/useWizardNavigation";
import { useWizardDoor } from "@/hooks/useWizardStepOrder";
import { needsMessage, wizardStepLabel, type NeedsStep } from "@/models/ProjectWizard";

interface WizardStepNeedsProps {
  needs: NeedsStep;
  onBack: () => void;
  onSkip?: (() => void) | undefined;
}

// A step opened before the one it builds on: one line on what is missing and a jump there, a signal rather than a locked rung.
export const WizardStepNeeds = ({ needs, onBack, onSkip }: WizardStepNeedsProps) => {
  const goTo = useWizardGoTo();
  const door = useWizardDoor();
  return (
    <div>
      <p className="text-sm text-muted-foreground">{needsMessage[needs]}</p>
      <WizardFooter onBack={onBack} skip={onSkip && <WizardSkipLink onClick={onSkip} />}>
        <Button type="button" onClick={() => goTo(needs)}>
          Go to {wizardStepLabel(needs, door)}
        </Button>
      </WizardFooter>
    </div>
  );
};
