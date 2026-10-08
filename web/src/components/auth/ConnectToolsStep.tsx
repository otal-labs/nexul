import { Button } from "@/components/ui/button";
import { ConnectorsSection } from "@/components/settings/ConnectorsSection";

interface ConnectToolsStepProps {
  onContinue: () => void;
}

// `bare` drops ConnectorsSection's own card chrome since WizardLayout already frames the title/subtitle.
export const ConnectToolsStep = ({ onContinue }: ConnectToolsStepProps) => (
  <div className="space-y-6">
    <ConnectorsSection bare />
    <Button className="w-full" onClick={onContinue}>
      Continue
    </Button>
  </div>
);
