import { WELCOME_STEPS, WELCOME_STEP_NAMES, type WelcomeStep } from "@nexul/client-core/welcome";

import { Switch } from "@/components/ui/switch";
import { useRoleDraft } from "@/hooks/useRoleDraft";
import { WELCOME_STEP_LINES } from "@/models/RoleSection";

interface RoleWelcomeRowProps {
  step: WelcomeStep;
}

// One Welcome wizard step as a switch; the draft keeps the wizard's order whatever order steps are ticked in.
export const RoleWelcomeRow = ({ step }: RoleWelcomeRowProps) => {
  const { steps, setSteps } = useRoleDraft();
  return (
    <li>
      <label className="flex cursor-pointer items-center gap-3 px-3 py-2.5">
        <span className="min-w-0 flex-1">
          <span className="block text-sm font-medium">{WELCOME_STEP_NAMES[step]}</span>
          <span className="block text-xs text-muted-foreground">{WELCOME_STEP_LINES[step]}</span>
        </span>
        <Switch
          checked={steps.includes(step)}
          onCheckedChange={(on) => setSteps(WELCOME_STEPS.filter((key) => (key === step ? on : steps.includes(key))))}
        />
      </label>
    </li>
  );
};
