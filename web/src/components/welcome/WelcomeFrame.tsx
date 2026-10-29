import { Button } from "@/components/ui/button";
import { WelcomeRail } from "@/components/welcome/WelcomeRail";
import { WelcomeStepPanel } from "@/components/welcome/WelcomeStepPanel";
import { WizardShell } from "@/components/wizard/WizardShell";
import { useWelcomeFlow } from "@/hooks/useWelcomeFlow";

export const WelcomeFrame = () => {
  const { leave, leaving } = useWelcomeFlow();
  return (
    <WizardShell
      rail={<WelcomeRail />}
      exit={
        <Button variant="ghost" size="sm" className="-mr-2 h-7 px-2 text-muted-foreground" loading={leaving} onClick={() => leave("/")}>
          Finish later
        </Button>
      }
    >
      <WelcomeStepPanel />
    </WizardShell>
  );
};
