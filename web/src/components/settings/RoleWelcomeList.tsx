import { WELCOME_STEPS } from "@nexul/client-core/welcome";

import { RoleWelcomeRow } from "@/components/settings/RoleWelcomeRow";
import { SettingsHint } from "@/components/settings/SettingsHint";

// The Welcome tab's body: what the role's holders are asked to set up, one switch per step.
export const RoleWelcomeList = () => (
  <div className="space-y-3">
    <SettingsHint>
      What the Welcome wizard asks someone holding this role to set up. A step gives no permission; switching one on asks
      everyone holding the role again.
    </SettingsHint>
    <ul className="divide-y divide-border overflow-hidden rounded-md border border-input">
      {WELCOME_STEPS.map((step) => (
        <RoleWelcomeRow key={step} step={step} />
      ))}
    </ul>
  </div>
);
