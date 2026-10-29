import { useState } from "react";

import { Button } from "@/components/ui/button";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { wizardOptionClass } from "@/components/wizard/WizardCardList";
import { WizardFooter } from "@/components/wizard/WizardFooter";
import { scanSummary } from "@/models/Repository";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import { cn } from "@/lib/utils";

type DeployChoice = "now" | "later";

interface DeployChoiceRowProps {
  value: DeployChoice;
  label: string;
  detail: string;
  mono?: boolean;
}

const DeployChoiceRow = ({ value, label, detail, mono = false }: DeployChoiceRowProps) => (
  <label htmlFor={`deploy-${value}`} className={wizardOptionClass}>
    <RadioGroupItem id={`deploy-${value}`} value={value} className="mt-0.5" />
    <span className="min-w-0">
      <span className="block text-sm font-medium">{label}</span>
      <span className={cn("mt-0.5 block text-sm text-muted-foreground", mono && "font-mono text-xs")}>{detail}</span>
    </span>
  </label>
);

interface WizardDeployStepProps {
  onNow: () => void;
  onLater: () => void;
  onBack: () => void;
}

// The attached repository was read and has something to build: deploy it now, or finish setup and deploy it later.
export const WizardDeployStep = ({ onNow, onLater, onBack }: WizardDeployStepProps) => {
  const repository = useProjectWizardStore((s) => s.repository);
  const scanResult = useProjectWizardStore((s) => s.scanResult);
  const candidate = useProjectWizardStore((s) => s.candidate);
  const [choice, setChoice] = useState<DeployChoice>("now");
  const branch = scanResult?.default_branch ?? repository?.default_branch;
  const found = [repository?.full_name, branch, scanSummary(candidate ?? undefined)].filter(Boolean).join(" · ");

  return (
    <div>
      <RadioGroup
        aria-label="Deploy it now?"
        value={choice}
        onValueChange={(value) => setChoice(value as DeployChoice)}
        className="gap-2"
      >
        <DeployChoiceRow value="now" label="Deploy now" detail={found} mono />
        <DeployChoiceRow value="later" label="Later" detail="Finish setup now and deploy from Settings → Services." />
      </RadioGroup>
      <WizardFooter onBack={onBack}>
        <Button type="button" onClick={choice === "now" ? onNow : onLater}>
          Continue
        </Button>
      </WizardFooter>
    </div>
  );
};
