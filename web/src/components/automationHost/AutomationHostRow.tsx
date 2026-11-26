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

// The name over its facts in mono, then the state as a dot beside its word, last seen only while offline as a runner words it; then the row's menu.
export const AutomationHostRow = ({ host, machine }: AutomationHostRowProps) => (
  <li className="grid grid-cols-[minmax(0,1fr)_auto_1.75rem] items-start gap-x-4 px-4 py-2.5">
    <div className="min-w-0">
      <span title={host.name} className={cn("block truncate text-sm leading-5 font-medium", !host.connected && "text-muted-foreground")}>{host.name}</span>
      <p className="mt-0.5 truncate font-mono text-xs text-muted-foreground">
        {host.machine && <span title="Machine">{host.machine}</span>}
        {host.machine && <span aria-hidden className="font-sans text-muted-foreground/40"> · </span>}
        {host.version || <span className="font-sans">unknown version</span>}
      </p>
      <AutomationHostManagers managers={host.managers} />
    </div>
    <span className="flex h-5 items-center gap-2 text-row text-muted-foreground tabular-nums">
      <RunnerStatusDot connected={host.connected} />
      {host.connected && "online"}
      {!host.connected && `offline · seen ${formatRelativeTime(host.last_seen)}`}
    </span>
    <span className="flex h-5 items-center justify-center">
      <AutomationHostMenu host={host} machine={machine} />
    </span>
  </li>
);
