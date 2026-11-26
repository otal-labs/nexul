import { AutomationHostManagers } from "@/components/automationHost/AutomationHostManagers";
import { AutomationHostMenu } from "@/components/automationHost/AutomationHostMenu";
import { RunnerStatusDot } from "@/components/runner/RunnerStatusDot";
import type { AutomationHost } from "@/models/AutomationHost";
import type { Machine } from "@/models/Runner";
import { cn } from "@/lib/utils";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface AutomationHostRowProps {
  host: AutomationHost;
  machine?: Pick<Machine, "actions"> | undefined;
}

// The name over its facts, then the state as a dot beside its word; then the row's menu.
export const AutomationHostRow = ({ host, machine }: AutomationHostRowProps) => (
  <li className="grid grid-cols-[minmax(0,1fr)_auto_1.75rem] items-start gap-x-4 px-4 py-2.5">
    <div className="min-w-0">
      <span title={host.name} className={cn("block truncate text-sm leading-5 font-medium", !host.connected && "text-muted-foreground")}>{host.name}</span>
      <p className="mt-0.5 truncate text-xs text-muted-foreground tabular-nums">
        {host.machine && <span title="Machine">{host.machine}</span>}
        {host.machine && " · "}
        <span className="font-mono">{host.version || "unknown version"}</span> · seen {formatRelativeTime(host.last_seen)}
      </p>
      <AutomationHostManagers managers={host.managers} />
    </div>
    <span className="flex h-5 items-center gap-2 text-row text-muted-foreground">
      <RunnerStatusDot connected={host.connected} />
      {host.connected && "online"}
      {!host.connected && "offline"}
    </span>
    <span className="flex h-5 items-center justify-center">
      <AutomationHostMenu host={host} machine={machine} />
    </span>
  </li>
);
