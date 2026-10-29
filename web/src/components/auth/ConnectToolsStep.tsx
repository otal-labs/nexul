import { Button } from "@/components/ui/button";
import { ConnectorsSection } from "@/components/settings/ConnectorsSection";
import { WizardFooter } from "@/components/wizard/WizardFooter";
import { useOwnerWizardBack } from "@/hooks/useOwnerWizardBack";

interface ConnectToolsStepProps {
  onContinue: () => void;
}

// `bare` drops ConnectorsSection's own card chrome: the wizard's step header already titles it.
export const ConnectToolsStep = ({ onContinue }: ConnectToolsStepProps) => {
  const back = useOwnerWizardBack();
  return (
    <div className="space-y-6">
      <ConnectorsSection bare />
      <WizardFooter onBack={back}>
        <Button onClick={onContinue}>Continue</Button>
      </WizardFooter>
    </div>
  );
};
