import { WizardRail } from "@/components/wizard/WizardRail";
import { WizardRailRow } from "@/components/wizard/WizardRailRow";
import { useWelcomeFlow } from "@/hooks/useWelcomeFlow";
import { STEP_NAMES } from "@/models/WelcomeWizard";

// Every step is a button, whatever happened there, so any step opens at any time.
export const WelcomeRail = () => {
  const { steps, marks, view, open } = useWelcomeFlow();
  const done = steps.filter((step) => marks[step] === "done").length;
  return (
    <WizardRail ariaLabel="Welcome steps" label="Welcome" done={done} total={steps.length}>
      {steps.map((step, i) => (
        <WizardRailRow
          key={step}
          label={STEP_NAMES[step]}
          mark={step === view ? "current" : marks[step]}
          ordinal={String(i + 1).padStart(2, "0")}
          total={steps.length}
          onOpen={() => open(step)}
          last={i === steps.length - 1}
        />
      ))}
    </WizardRail>
  );
};
