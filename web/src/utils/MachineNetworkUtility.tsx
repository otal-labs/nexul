import type { Container, Stack } from "@/models/Stack";

export interface MachineNetwork {
  name: string;
  services: string[];
}

const builtinNetworks = new Set(["bridge", "host", "none"]);

// Every docker network the runner has seen a container join on machine, with the services on it, sorted by name.
// always is listed even before anything on it has been observed (a stack whose first deploy is still running).
export const machineNetworks = (
  stacks: Stack[],
  containers: Container[],
  machine: string,
  always: string,
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
    .map(([name, services]) => ({ name, services: [...services].sort() }))
    .sort((a, b) => a.name.localeCompare(b.name));
};

export const networkLabel = (network: MachineNetwork): string =>
  network.services.length > 0 ? `${network.name} — ${network.services.join(", ")}` : network.name;
