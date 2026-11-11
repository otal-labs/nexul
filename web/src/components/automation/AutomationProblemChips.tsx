import { KeyRound } from "lucide-react";
import { Link } from "react-router";

import { AutomationAttentionChip } from "@/components/automation/AutomationAttentionChip";
import { NoFillBadge } from "@/components/ui/badge";
import { useFetchAutomationHosts, useOfflineAutomationHostIds } from "@/hooks/AutomationHostHooks";
import { useTabPath } from "@/hooks/useTabPath";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { AutomationAttention, automationAttentions, automationLastRunFailed, automationPoolUnresolvable, type Automation } from "@/models/Automation";
import { runOutcomes } from "@/models/AutomationRun";

const chipLinkClass = "group inline-flex rounded-sm";
// Ink and a dotted underline set a problem apart from the plain facts before it, and say it can be followed.
const chipClass = "text-foreground underline decoration-muted-foreground/50 decoration-dotted underline-offset-4 group-hover:decoration-solid group-hover:decoration-current";

interface AutomationProblemChipsProps {
  automation: Automation;
}

// Every open problem on the meta line, each a link to where it is fixed.
export const AutomationProblemChips = ({ automation }: AutomationProblemChipsProps) => {
  const offlineHostIds = useOfflineAutomationHostIds();
  const { data: hosts } = useFetchAutomationHosts();
  const { tabPath } = useTabPath();
  const wsPath = useWorkspacePath();
  const fixedAt: Record<AutomationAttention, string> = {
    [AutomationAttention.HostOffline]: wsPath("/automations/hosts"),
    [AutomationAttention.NeedsConfiguration]: tabPath("settings"),
    [AutomationAttention.TokenRevoked]: tabPath("settings"),
    [AutomationAttention.PendingVersion]: tabPath("versions"),
  };
  const lastRun = automation.last_run && runOutcomes[automation.last_run.outcome];

  return (
    <>
      {automation.enabled && lastRun && automationLastRunFailed(automation) && (
        <Link to={tabPath()} className={chipLinkClass}>
          <NoFillBadge icon={lastRun.icon} color="text-destructive" className={`whitespace-nowrap ${chipClass}`}>
            Last run {lastRun.past}
          </NoFillBadge>
        </Link>
      )}
      {automation.enabled && automationPoolUnresolvable(automation, hosts) && (
        <Link to={wsPath("/automations/hosts")} className={chipLinkClass}>
          <NoFillBadge icon={KeyRound} color="text-warning" className={`whitespace-nowrap ${chipClass}`}>
            Host can&apos;t resolve secret references
          </NoFillBadge>
        </Link>
      )}
      {automationAttentions(automation, offlineHostIds).map((attention) => (
        <Link key={attention} to={fixedAt[attention]} className={chipLinkClass}>
          <AutomationAttentionChip automation={automation} attention={attention} className={chipClass} />
        </Link>
      ))}
    </>
  );
};
