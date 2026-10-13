import { Button } from "@/components/ui/button";
import { WelcomeStepNode } from "@/components/welcome/WelcomeStepNode";
import { useWelcomeFlow } from "@/hooks/useWelcomeFlow";
import { MARK_WORD, STEP_NAMES, type WizardStep } from "@/models/WelcomeWizard";

interface WelcomeLaterRowProps {
  step: WizardStep;
}

export const WelcomeLaterRow = ({ step }: WelcomeLaterRowProps) => {
  const { marks, open } = useWelcomeFlow();
  return (
    <li className="flex items-center gap-3 py-2.5">
      <WelcomeStepNode state={marks[step]} />
      <span className="flex-1 text-sm">{STEP_NAMES[step]}</span>
      <span className="font-mono text-xs text-muted-foreground">{MARK_WORD[marks[step]]}</span>
      <Button variant="ghost" size="sm" onClick={() => open(step)} aria-label={`Open ${STEP_NAMES[step]}`}>
        Open
      </Button>
    </li>
  );
};
