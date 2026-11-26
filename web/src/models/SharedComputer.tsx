import type { Computer } from "@/models/Pairing";

// Whose a computer shared with the viewer is, and what its owner's rule lets them do there (ADR 0148).
export interface SharedComputer {
  owner_id: string;
  owner_name: string;
  permissions: string[];
  // Whether the viewer put it in their own run order, at its position there (ADR 0211).
  in_order?: boolean;
}

export const ComputerPermission = {
  See: "computers:see",
  RunAgents: "computers:run_agents",
  RunCommands: "computers:run_commands",
  ReadLogs: "computers:read_logs",
} as const;

// How a person a computer is shared with reads it.
export const sharedComputerName = (computer: Computer) =>
  computer.shared ? `${computer.shared.owner_name}'s ${computer.name} (shared)` : computer.name;

// What the viewer may do on a shared computer, in words; seeing it alone says so.
export const sharedComputerAbilities = (shared: SharedComputer) => {
  const abilities = [
    shared.permissions.includes(ComputerPermission.RunAgents) && "Run agents",
    shared.permissions.includes(ComputerPermission.RunCommands) && "Run commands",
    shared.permissions.includes(ComputerPermission.ReadLogs) && "Read logs",
  ].filter(Boolean);
  return abilities.length > 0 ? abilities.join(" · ") : "See only";
};

// Why a picker lists a shared computer it can't pick: its owner's share doesn't give Run agents.
export const notRunnableYet = "can't run agents here";

// Whether the viewer's runs may land on a computer shared with them: its owner's share gives Run agents.
export const runsAgents = (computer: Computer) => !!computer.shared?.permissions.includes(ComputerPermission.RunAgents);

// The two halves of the one computers request: the viewer's own, and those shared with them.
export const ownComputers = (computers: Computer[]) => computers.filter((c) => !c.shared);
export const sharedComputers = (computers: Computer[]) => computers.filter((c) => !!c.shared);

// The viewer's run order: their own computers and the shared ones they added, by position, their own first on a tie.
export const orderComputers = (computers: Computer[]) =>
  computers.filter((c) => !c.shared || c.shared.in_order).sort((a, b) => a.position - b.position || Number(!!a.shared) - Number(!!b.shared));
