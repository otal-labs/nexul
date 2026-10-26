import { TicketIcon } from "lucide-react";

import { ActionSwitch } from "@/components/action/ActionSwitch";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { SettingsHint } from "@/components/settings/SettingsHint";
import { StackCrashNoticesCard } from "@/components/stack/StackCrashNoticesCard";
import { StackErrorRateCard } from "@/components/stack/StackErrorRateCard";
import { useUpdateStack } from "@/hooks/StackHooks";
import { useStackActions } from "@/hooks/useStackActions";
import type { Stack } from "@/models/Stack";

interface StackAlertsSectionProps {
  stack: Stack;
}

// Saving sends the whole stack, so a reader whose env values were withheld must not flip it and blank them.
const reasonOff = (stack: Stack): string | undefined => {
  if (!stack.project_id) return "Needs a project";
  if (stack.env_masked) return "Your role can't see this stack's variables";
  return undefined;
};

export const StackAlertsSection = ({ stack }: StackAlertsSectionProps) => {
  const answer = useStackActions(stack);
  const update = useUpdateStack();
  const off = reasonOff(stack);

  return (
    <>
      <SettingsCard id="crash-tickets" title="Crash tickets" icon={TicketIcon}>
        <div className="flex items-start gap-3">
          <div className="min-w-0 flex-1 space-y-0.5">
            <p className="text-sm font-medium">
              Open a ticket when a service crashes
            </p>
            <SettingsHint>
              {off ?? "One ticket per service in the project's first stage, created by the instance. A repeat or a recovery adds a comment. The error-rate rule opens one the same way."}
            </SettingsHint>
          </div>
          <ActionSwitch
            permission="stacks:write"
            projectId={stack.project_id || undefined}
            answer={answer("stacks:write")}
            verb="change this stack's alerts"
            checked={!!stack.crash_tickets}
            disabled={!!off || update.isPending}
            onCheckedChange={(crash_tickets) => update.mutate({ ...stack, crash_tickets })}
            aria-label="Open a ticket when a service crashes"
            {...(off && { title: off })}
            className="mt-0.5"
          />
        </div>
      </SettingsCard>
      <StackCrashNoticesCard stack={stack} off={off} />
      <StackErrorRateCard stack={stack} />
    </>
  );
};
