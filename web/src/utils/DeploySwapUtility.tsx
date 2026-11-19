import { parseContainerPort, type Container } from "@/models/Stack";

// Why a deploy restarts a service in place instead of swapping it, in the runner's order of checks.
export const SwapBlocker = {
  NoHostname: "no hostname",
  HostPort: "host port",
  NoHealthcheck: "no healthcheck",
} as const;

export type SwapBlocker = (typeof SwapBlocker)[keyof typeof SwapBlocker];

export interface DeployModes {
  swapped: string[];
  restarted: { reason: SwapBlocker; names: string[] }[];
}

// undefined when a deploy swaps the service, null when it cannot tell yet: no runner has said whether it has a healthcheck.
const blockerOf = (c: Container, hostnames: string[]): SwapBlocker | null | undefined => {
  if (hostnames.length === 0) return SwapBlocker.NoHostname;
  if ((c.ports ?? []).some((p) => !!parseContainerPort(p)?.host)) return SwapBlocker.HostPort;
  if (c.healthcheck === undefined) return null;
  if (!c.healthcheck) return SwapBlocker.NoHealthcheck;
  return undefined;
};

// How the next deploy treats each service: swapped with no downtime, or restarted in place grouped by why.
export const deployModes = (services: Container[], hostnamesOf: (id: string) => string[]): DeployModes => {
  const swapped: string[] = [];
  const restarted = new Map<SwapBlocker, string[]>();
  for (const c of services) {
    const blocker = blockerOf(c, hostnamesOf(c.id));
    if (blocker === undefined) swapped.push(c.name);
    if (blocker) restarted.set(blocker, [...(restarted.get(blocker) ?? []), c.name]);
  }
  return { swapped, restarted: [...restarted].map(([reason, names]) => ({ reason, names })) };
};
