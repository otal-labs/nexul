import { Cable, SquareTerminal } from "lucide-react";

import { CheckRow } from "@/components/pairing/CheckRow";
import type { Check } from "@/models/ComputerChecks";
import { tunnelConnected, tunnelOnline, type TunnelStatus } from "@/models/Pairing";

const tunnelCheck = (status: TunnelStatus): Check => {
  const base = { name: "Tunnel online" };
  if (status.tunnel === "healthy") return { ...base, detail: "cloudflared reached Cloudflare", state: "passed", label: "Online" };
  if (status.tunnel === "degraded") return { ...base, detail: "Some connections are unhealthy", state: "warning", label: "Degraded" };
  if (status.tunnel === "down") return { ...base, detail: "cloudflared lost its connections", state: "failed", label: "Down" };
  return { ...base, detail: "Waiting for cloudflared to connect", state: "waiting", label: "Waiting" };
};

const harnessCheck = (status: TunnelStatus): Check => {
  const base = { name: "T3 Code answering" };
  if (status.harness_reachable) {
    return { ...base, detail: `T3 Code ${status.harness_version ?? ""}`.trim(), state: "passed", label: "Answering" };
  }
  if (tunnelOnline(status)) return { ...base, detail: "Start T3 Code on this computer", state: "waiting", label: "Waiting" };
  return { ...base, detail: "Checked once the tunnel is online", state: "waiting", label: "Waiting" };
};

interface TunnelChecksProps {
  status: TunnelStatus;
}

// The two checks pairing waits on: Cloudflare sees the connector, and the harness answers through the hostname.
export const TunnelChecks = ({ status }: TunnelChecksProps) => (
  <div className="rounded-lg border border-border bg-card" role="status" aria-live="polite">
    <div className="flex items-center justify-between border-b border-border px-3 py-2">
      <span className="text-xs font-semibold">Checks</span>
      <span className="text-xs text-muted-foreground">{tunnelConnected(status) ? "Both passed" : "Waiting"}</span>
    </div>
    <ul className="divide-y divide-border">
      <CheckRow icon={Cable} check={tunnelCheck(status)} />
      <CheckRow icon={SquareTerminal} check={harnessCheck(status)} />
    </ul>
  </div>
);
