import { ActionButton } from "@/components/action/ActionButton";
import { AutomationVersionCode } from "@/components/automation/AutomationVersionCode";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { useMergeAutomationVersion } from "@/hooks/AutomationVersionHooks";
import type { AutomationVersion } from "@/models/AutomationVersion";

interface AutomationVersionDiffProps {
  automationId: string;
  pending: AutomationVersion;
  active: AutomationVersion | null;
}

export const AutomationVersionDiff = ({ automationId, pending, active }: AutomationVersionDiffProps) => {
  const merge = useMergeAutomationVersion(automationId);
  // A source diffed against a bundle would mark every line, so the pending file then shows whole.
  const against = active && !!active.source === !!pending.source ? active.sequence : undefined;

  return (
    <SettingsCard
      id="pending-version"
      title={`#${pending.sequence} pending`}
      description={
        <>
          {pending.message}
          {against === undefined && active && <span className="block">Whole file, no diff</span>}
        </>
      }
      aside={
        <ActionButton
          permission="automations:write"
          verb="merge automation versions"
          type="button"
          size="sm"
          onClick={() => merge.mutate(pending.id)}
          loading={merge.isPending}
        >
          Merge #{pending.sequence}
        </ActionButton>
      }
    >
      <AutomationVersionCode automationId={automationId} sequence={pending.sequence} against={against} />
    </SettingsCard>
  );
};
