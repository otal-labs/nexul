import { WizardRail } from "@/components/wizard/WizardRail";
import { WizardRailRow } from "@/components/wizard/WizardRailRow";
import { OWNER_STEPS } from "@/models/OwnerWizard";
import type { WizardRailMark } from "@/models/WizardRail";

const markOf = (index: number, step: number): WizardRailMark => {
  if (index + 1 < step) return "done";
  if (index + 1 === step) return "current";
  return "left";
};

interface OwnerWizardRailProps {
  step: number;
}

// No row opens a step: the workspace step makes the owner the tools step needs, so the flow only moves through its bar.
export const OwnerWizardRail = ({ step }: OwnerWizardRailProps) => (
  <WizardRail ariaLabel="Owner setup steps" label="Get started" done={step - 1} total={OWNER_STEPS.length}>
    {OWNER_STEPS.map((s, i) => (
      <WizardRailRow
        key={s.label}
        label={s.label}
        mark={markOf(i, step)}
        ordinal={String(i + 1).padStart(2, "0")}
        total={OWNER_STEPS.length}
        last={i === OWNER_STEPS.length - 1}
      />
    ))}
  </WizardRail>
);
