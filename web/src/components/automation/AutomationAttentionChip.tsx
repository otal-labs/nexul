import { AlertTriangle, Clock, KeyRound, Unplug, type LucideIcon } from "lucide-react";

import { NoFillBadge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { AutomationAttention, type Automation } from "@/models/Automation";

const chips: Record<AutomationAttention, { icon: LucideIcon; label: (automation: Automation) => string; mono?: boolean }> = {
  [AutomationAttention.HostOffline]: { icon: Unplug, label: () => "Host offline" },
  [AutomationAttention.NeedsConfiguration]: { icon: AlertTriangle, label: () => "Needs configuration" },
  [AutomationAttention.TokenRevoked]: { icon: KeyRound, label: () => "Token revoked" },
  [AutomationAttention.PendingVersion]: { icon: Clock, label: (automation) => `#${automation.pending_version} pending`, mono: true },
};

interface AutomationAttentionChipProps {
  automation: Automation;
  attention: AutomationAttention | null;
  /** Muted while a louder state (a failing last run) already leads the row. */
  muted?: boolean;
  className?: string;
}

// The amber mark only while the row is in Needs attention, so the marked chips and that count name the same set; otherwise a plain muted note.
export const AutomationAttentionChip = ({ automation, attention, muted = false, className }: AutomationAttentionChipProps) => {
  if (!attention) return null;
  const chip = chips[attention];
  const chipClass = cn("shrink-0 items-baseline leading-5 whitespace-nowrap", chip.mono && "font-mono", className);
  if (!automation.enabled || muted) {
    return <span className={cn("inline-flex text-xs text-muted-foreground", chipClass)}>{chip.label(automation)}</span>;
  }
  return (
    <NoFillBadge icon={chip.icon} iconClassName="self-center" color="text-warning" className={chipClass}>
      {chip.label(automation)}
    </NoFillBadge>
  );
};
