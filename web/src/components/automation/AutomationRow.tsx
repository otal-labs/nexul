import { Link } from "react-router";

import { AutomationAttentionChip } from "@/components/automation/AutomationAttentionChip";
import { AutomationStatusMark } from "@/components/automation/AutomationStatusMark";
import { ActionSwitch } from "@/components/action/ActionSwitch";
import { AutomationRunOutcome } from "@/enums/Automation";
import { useSetAutomationEnabled } from "@/hooks/AutomationHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { cn } from "@/lib/utils";
import { AutomationState } from "@/models/Automation";
import { automationLastRunText, automationWeekText, type AutomationListItem } from "@/utils/AutomationListUtility";

interface AutomationRowProps {
  item: AutomationListItem;
}

// Two lines: the name and its one chip, then why it is failing (the last error) or, when it isn't, what it does.
export const AutomationRow = ({ item: { automation, state, attention } }: AutomationRowProps) => {
  const setEnabled = useSetAutomationEnabled();
  const wsPath = useWorkspacePath();
  const failing = state === AutomationState.Failing;
  const crashed = automation.last_run?.outcome === AutomationRunOutcome.Crash;
  const week = automationWeekText(automation);

  return (
    <li className="grid grid-cols-[1.25rem_minmax(0,1fr)_auto_2.25rem] items-baseline gap-x-3 px-4 py-2.5 transition-colors duration-150 ease-standard hover:bg-accent">
      <span className="row-span-2 self-start">
        <AutomationStatusMark automation={automation} />
      </span>
      <div className="flex h-5 min-w-0 items-baseline gap-2">
        <Link
          to={wsPath(`/automations/${automation.id}`)}
          title={automation.name}
          className={cn("min-w-0 truncate text-sm font-medium hover:underline", !automation.enabled && "text-muted-foreground")}
        >
          {automation.name}
        </Link>
        <AutomationAttentionChip automation={automation} attention={attention} />
      </div>
      <span className="font-mono text-xs leading-5 whitespace-nowrap text-muted-foreground tabular-nums" title="The last run, then the runs of the last seven days">
        {automationLastRunText(automation)}
        {week && <span className="max-lg:hidden"> · {week}</span>}
      </span>
      <span className="row-span-2 flex h-5 items-center self-start justify-self-end">
        <ActionSwitch
          permission="automations:write"
          verb="switch automations on or off"
          aria-label={automation.enabled ? `Disable ${automation.name}` : `Enable ${automation.name}`}
          checked={automation.enabled}
          disabled={setEnabled.isPending}
          onCheckedChange={(checked) => setEnabled.mutate({ id: automation.id, enabled: checked })}
        />
      </span>
      {failing && (
        <p className="col-span-2 col-start-2 truncate font-mono text-xs leading-5 text-muted-foreground" title={automation.last_run?.error}>
          <span className="text-destructive">{crashed ? "Crashed" : "Failed"}</span>
          {automation.last_run?.error && `: ${automation.last_run.error}`}
        </p>
      )}
      {!failing && automation.description && (
        <p className="col-span-2 col-start-2 truncate text-[13px] leading-5 text-muted-foreground" title={automation.description}>
          {automation.description}
        </p>
      )}
      {!failing && !automation.description && <p className="col-span-2 col-start-2 text-[13px] leading-5 text-muted-foreground italic">No description</p>}
    </li>
  );
};
