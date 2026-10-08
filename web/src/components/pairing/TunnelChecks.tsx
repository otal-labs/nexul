import { Cable, SquareTerminal, type LucideIcon } from "lucide-react";

import { tunnelConnected, tunnelOnline, type TunnelStatus } from "@/models/Pairing";
import { cn } from "@/lib/utils";

type CheckState = "waiting" | "passed" | "warning" | "failed";

interface Check {
  icon: LucideIcon;
  name: string;
  detail: string;
  state: CheckState;
  label: string;
}

// Status is the only colour on the dialog; waiting stays monochrome and pulses.
const STATE_DOT: Record<CheckState, string> = {
  waiting: "bg-muted-foreground/60 animate-[status-pulse_2.4s_ease-standard_infinite]",
  passed: "bg-success",
  warning: "bg-warning",
  failed: "bg-destructive",
};

const tunnelCheck = (status: TunnelStatus): Check => {
  const base = { icon: Cable, name: "Tunnel online" };
  if (status.tunnel === "healthy") return { ...base, detail: "cloudflared reached Cloudflare", state: "passed", label: "Online" };
  if (status.tunnel === "degraded") return { ...base, detail: "Some connections are unhealthy", state: "warning", label: "Degraded" };
  if (status.tunnel === "down") return { ...base, detail: "cloudflared lost its connections", state: "failed", label: "Down" };
  return { ...base, detail: "Waiting for cloudflared to connect", state: "waiting", label: "Waiting" };
};

const harnessCheck = (status: TunnelStatus): Check => {
  const base = { icon: SquareTerminal, name: "T3 Code answering" };
  if (status.harness_reachable) {
    return { ...base, detail: `T3 Code ${status.harness_version ?? ""}`.trim(), state: "passed", label: "Answering" };
  }
  if (tunnelOnline(status)) return { ...base, detail: "Start T3 Code on this computer", state: "waiting", label: "Waiting" };
  return { ...base, detail: "Checked once the tunnel is online", state: "waiting", label: "Waiting" };
};

interface TunnelCheckRowProps {
  check: Check;
}

const TunnelCheckRow = ({ check }: TunnelCheckRowProps) => {
  const Icon = check.icon;
  return (
    <li className="flex items-center gap-2.5 px-3 py-2.5">
      <span className="grid size-7 shrink-0 place-items-center rounded-md border border-border bg-background text-muted-foreground">
        <Icon className="size-3.5" aria-hidden />
      </span>
      <span className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span className="truncate text-xs font-medium">{check.name}</span>
        <span className="truncate font-mono text-[11px] text-muted-foreground">{check.detail}</span>
      </span>
      <span className="flex shrink-0 items-center gap-1.5 text-[11px] font-medium text-muted-foreground">
        <span className={cn("size-1.5 rounded-full transition-colors duration-150 ease-standard", STATE_DOT[check.state])} aria-hidden />
        {check.label}
      </span>
    </li>
  );
};

interface TunnelChecksProps {
  status: TunnelStatus;
}

// The two checks pairing waits on: Cloudflare sees the connector, and the harness answers through the hostname.
export const TunnelChecks = ({ status }: TunnelChecksProps) => (
  <div className="rounded-lg border border-border bg-card" role="status" aria-live="polite">
    <div className="flex items-center justify-between border-b border-border px-3 py-2">
      <span className="text-xs font-semibold">Checks</span>
      <span className="text-[11px] text-muted-foreground">{tunnelConnected(status) ? "Both passed" : "Waiting"}</span>
    </div>
    <ul className="divide-y divide-border">
      <TunnelCheckRow check={tunnelCheck(status)} />
      <TunnelCheckRow check={harnessCheck(status)} />
    </ul>
  </div>
);
