import { WizardFooter } from "@/components/wizard/WizardFooter";
import { Button } from "@/components/ui/button";

interface WizardProjectCardProps {
  name: string;
  prefix: string;
  onContinue: () => void;
  onBack: () => void;
}

// The project this run made: its name over its prefix, and the one way on.
export const WizardProjectCard = ({ name, prefix, onContinue, onBack }: WizardProjectCardProps) => (
  <div>
    <div className="flex items-center gap-3 rounded-lg border border-border bg-card px-3 py-2.5">
      <span className="min-w-0 flex-1">
        <span className="block truncate text-sm font-medium">{name}</span>
        <span className="block truncate font-mono text-xs text-muted-foreground">{prefix}</span>
      </span>
      <span className="shrink-0 text-xs text-muted-foreground">Rename in Settings</span>
    </div>
    <WizardFooter onBack={onBack}>
      <Button type="button" onClick={onContinue}>
        Continue
      </Button>
    </WizardFooter>
  </div>
);
