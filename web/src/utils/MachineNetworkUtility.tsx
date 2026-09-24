import type { Container, Stack } from "@/models/Stack";

export interface MachineNetwork {
  name: string;
  services: string[];
  // A hostname on a branch deployment is served by the gateway whose home network this is (deploy's gateway check).
  hasGateway: boolean;
}

const builtinNetworks = new Set(["bridge", "host", "none"]);

// Observed docker networks with their services, sorted; `always` is listed even before its first deploy lands.
export const machineNetworks = (
  stacks: Stack[],
  containers: Container[],
  machine: string,
  always: string,
  gatewayNetworks: Set<string>,
): MachineNetwork[] => {
  const onMachine = new Set(stacks.filter((s) => s.machine === machine).map((s) => s.id));
  const byName = new Map<string, Set<string>>([[always, new Set()]]);
  for (const c of containers) {
    if (!onMachine.has(c.stack_id)) continue;
    for (const n of c.networks ?? []) {
      if (builtinNetworks.has(n.name)) continue;
      byName.set(n.name, (byName.get(n.name) ?? new Set()).add(c.name));
    }
  }
  return [...byName]
    .map(([name, services]) => ({ name, services: [...services].sort(), hasGateway: gatewayNetworks.has(name) }))
    .sort((a, b) => a.name.localeCompare(b.name));
};

export const noGatewayReason = "no gateway, hostnames unavailable";

export const networkLabel = (network: MachineNetwork): string => {
  const base = network.services.length > 0 ? `${network.name} — ${network.services.join(", ")}` : network.name;
  return network.hasGateway ? base : `${base} · ${noGatewayReason}`;
};
