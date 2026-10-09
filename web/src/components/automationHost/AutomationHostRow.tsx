import { RemoveAutomationHostButton } from "@/components/automationHost/RemoveAutomationHostButton";
import { RunnerStatusDot } from "@/components/runner/RunnerStatusDot";
import type { AutomationHost } from "@/models/AutomationHost";
import { cn } from "@/lib/utils";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface AutomationHostRowProps {
  host: AutomationHost;
}

// The runner row's columns: the connection dot, the name over its facts, and the state in words.
export const AutomationHostRow = ({ host }: AutomationHostRowProps) => (
  <li className="grid grid-cols-[1.25rem_minmax(0,1fr)_auto_2rem] items-center gap-x-3 px-4 py-2.5 transition-colors duration-150 ease-standard hover:bg-accent/40">
    <RunnerStatusDot connected={host.connected} />
    <div className="min-w-0">
      <span className={cn("block truncate font-mono text-sm", !host.connected && "text-muted-foreground")}>{host.name}</span>
      <p className="mt-0.5 truncate font-mono text-xs text-muted-foreground">
        {host.machine} · {host.version || "unknown version"} · seen {formatRelativeTime(host.last_seen)}
      </p>
    </div>
    <span className={cn("text-sm", host.connected ? "text-success" : "text-muted-foreground")}>
      {host.connected ? "online" : "offline"}
    </span>
    <RemoveAutomationHostButton host={host} />
  </li>
);
