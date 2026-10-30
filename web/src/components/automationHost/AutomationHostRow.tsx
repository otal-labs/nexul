import { RemoveAutomationHostButton } from "@/components/automationHost/RemoveAutomationHostButton";
import { RunnerStatusDot } from "@/components/runner/RunnerStatusDot";
import type { AutomationHost } from "@/models/AutomationHost";
import type { Machine } from "@/models/Runner";
import { cn } from "@/lib/utils";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface AutomationHostRowProps {
  host: AutomationHost;
  machine?: Pick<Machine, "actions"> | undefined;
}

// The runner row's columns: the connection dot, the name over its facts, and the state in words.
export const AutomationHostRow = ({ host, machine }: AutomationHostRowProps) => (
  <li className="grid grid-cols-[1.25rem_minmax(0,1fr)_auto_2rem] items-start gap-x-3 px-2 py-2.5 transition-colors duration-150 ease-standard hover:bg-accent">
    <span className="flex h-5 items-center justify-center">
      <RunnerStatusDot connected={host.connected} />
    </span>
    <div className="min-w-0">
      <span className={cn("block truncate text-sm font-medium", !host.connected && "text-muted-foreground")}>{host.name}</span>
      <p className="mt-0.5 truncate font-mono text-xs text-muted-foreground">
        {host.machine} · {host.version || "unknown version"} · seen {formatRelativeTime(host.last_seen)}
      </p>
    </div>
    <span className={cn("text-sm leading-5", host.connected ? "text-success" : "text-muted-foreground")}>
      {host.connected ? "online" : "offline"}
    </span>
    <span className="flex h-5 items-center justify-center">
      <RemoveAutomationHostButton host={host} machine={machine} />
    </span>
  </li>
);
