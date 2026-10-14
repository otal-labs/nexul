import type { Machine, Runner } from "@/models/Runner";

interface MachineDeleteDetailsProps {
  machine: Machine;
  runners: Runner[];
}

// What the confirm says goes with the machine: its record and rules, and the runners that are offline.
export const MachineDeleteDetails = ({ machine, runners }: MachineDeleteDetailsProps) => {
  const offline = runners.filter((runner) => !runner.connected);
  return (
    <ul className="space-y-1.5 border-y border-border py-3 text-sm">
      <li>
        The machine <span className="font-mono">{machine.name}</span> and the rules for who sees, deploys to, manages and deletes it
      </li>
      {offline.length > 0 && (
        <li>
          {offline.length === 1 ? "The offline runner" : "The offline runners"}{" "}
          <span className="font-mono break-all">{offline.map((runner) => runner.name).join(", ")}</span>, which{" "}
          {offline.length === 1 ? "uninstalls itself" : "uninstall themselves"} if {offline.length === 1 ? "it comes" : "they come"} back
        </li>
      )}
    </ul>
  );
};
