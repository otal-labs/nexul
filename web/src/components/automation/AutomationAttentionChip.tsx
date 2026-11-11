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

// Amber only while the row is in Needs attention, so the amber chips and that count name the same set.
export const AutomationAttentionChip = ({ automation, attention, muted = false, className }: AutomationAttentionChipProps) => {
  if (!attention) return null;
  const chip = chips[attention];
  return (
    <NoFillBadge
      icon={chip.icon}
      iconClassName="self-center"
      color={automation.enabled && !muted ? "text-warning" : "text-muted-foreground"}
      className={cn("shrink-0 items-baseline leading-5 whitespace-nowrap", chip.mono && "font-mono", className)}
    >
      {chip.label(automation)}
    </NoFillBadge>
  );
};
