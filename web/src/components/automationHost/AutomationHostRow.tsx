import { RemoveAutomationHostButton } from "@/components/automationHost/RemoveAutomationHostButton";
import { entranceDelayMs } from "@/components/runner/motion";
import { RunnerStatusBadge } from "@/components/runner/RunnerStatusBadge";
import type { AutomationHost } from "@/models/AutomationHost";
import { cn } from "@/lib/utils";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface AutomationHostRowProps {
  host: AutomationHost;
  index?: number;
}

// <li> owns hover, the inner <div> owns the entrance, so a refetch reusing key={host.id} won't replay it.
export const AutomationHostRow = ({ host, index = 0 }: AutomationHostRowProps) => (
  <li
    className={cn(
      "transition-colors duration-150 ease-standard hover:bg-accent/40",
      !host.connected && "opacity-60",
    )}
  >
    <div
      className="animate-in fade-in-0 slide-in-from-bottom-1 flex items-center gap-3 px-4 py-3 duration-150 ease-out"
      style={{ animationDelay: `${entranceDelayMs(index)}ms` }}
    >
      <RunnerStatusBadge connected={host.connected} />
      <div className="min-w-0 flex-1">
        <span className="block truncate font-mono font-medium">{host.name}</span>
        <p className="mt-0.5 truncate font-mono text-xs text-muted-foreground">
          {host.machine} · {host.version || "unknown version"} · last seen {formatRelativeTime(host.last_seen)}
        </p>
      </div>
      <RemoveAutomationHostButton host={host} />
    </div>
  </li>
);
