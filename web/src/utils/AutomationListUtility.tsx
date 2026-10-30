import { AutomationKind } from "@/enums/Automation";
import {
  AutomationState,
  automationAttention,
  automationState,
  type Automation,
  type AutomationAttention,
} from "@/models/Automation";
import { formatRelativeTime } from "@/utils/TimeUtility";

export interface AutomationListItem {
  automation: Automation;
  state: AutomationState;
  attention: AutomationAttention | null;
}

export interface AutomationGroup {
  label: string;
  items: AutomationListItem[];
}

const urgency: Record<AutomationState, number> = {
  [AutomationState.Failing]: 0,
  [AutomationState.Attention]: 1,
  [AutomationState.On]: 2,
  [AutomationState.Off]: 3,
};

export const automationListItems = (automations: readonly Automation[], offlineHostIds: ReadonlySet<string>): AutomationListItem[] =>
  automations
    .map((automation) => ({
      automation,
      state: automationState(automation, offlineHostIds),
      attention: automationAttention(automation, offlineHostIds),
    }))
    .sort((a, b) => urgency[a.state] - urgency[b.state] || a.automation.name.localeCompare(b.automation.name));

const mostUrgent = (items: readonly AutomationListItem[]) => Math.min(...items.map((item) => urgency[item.state]));

// Default before Custom unless Custom holds something more urgent, so a failing automation never sits under healthy ones.
export const automationGroups = (items: readonly AutomationListItem[]): AutomationGroup[] =>
  [
    { label: "Default", items: items.filter((item) => item.automation.kind === AutomationKind.Default) },
    { label: "Custom", items: items.filter((item) => item.automation.kind !== AutomationKind.Default) },
  ]
    .filter((group) => group.items.length > 0)
    .sort((a, b) => mostUrgent(a.items) - mostUrgent(b.items));

// The meta column's first part: when it last ran.
export const automationLastRunText = (automation: Automation): string =>
  automation.last_run ? formatRelativeTime(automation.last_run.started_at) : "never ran";

// The meta column's second part: the last seven days' runs, or null when there is nothing to add to "never ran".
export const automationWeekText = (automation: Automation): string | null => {
  const runs = automation.runs_7d;
  if (!runs || runs.total === 0) return automation.last_run ? "no runs · 7d" : null;
  if (runs.failed === 0) return `${runs.total} ${runs.total === 1 ? "run" : "runs"} · 7d`;
  return `${runs.failed} of ${runs.total} failed · 7d`;
};
