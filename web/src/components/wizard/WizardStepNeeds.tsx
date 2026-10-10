import { Button } from "@/components/ui/button";
import { WizardFooter } from "@/components/wizard/WizardFooter";
import { WizardSkipLink } from "@/components/wizard/WizardSkipLink";
import { useWizardGoTo } from "@/hooks/useWizardNavigation";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import { wizardStepLabel, type WizardStepId } from "@/models/ProjectWizard";

interface WizardStepNeedsProps {
  message: string;
  needs: WizardStepId;
  onBack: () => void;
  onSkip?: (() => void) | undefined;
}

// A step opened before the one it builds on: it says what is missing and jumps there, a signal rather than a locked rung.
export const WizardStepNeeds = ({ message, needs, onBack, onSkip }: WizardStepNeedsProps) => {
  const goTo = useWizardGoTo();
  const isAttach = useProjectWizardStore((s) => !!s.attachStackId);
  return (
    <div>
      <p className="text-sm text-muted-foreground">{message}</p>
      <WizardFooter onBack={onBack} skip={onSkip && <WizardSkipLink onClick={onSkip} />}>
        <Button type="button" onClick={() => goTo(needs)}>
          Go to {wizardStepLabel(needs, isAttach)}
        </Button>
      </WizardFooter>
    </div>
  );
};
