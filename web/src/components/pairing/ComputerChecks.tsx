import { Laptop, Link2, SquareTerminal } from "lucide-react";

import { CheckRow } from "@/components/pairing/CheckRow";
import { computerChecks, type Check } from "@/models/ComputerChecks";
import type { Computer } from "@/models/Pairing";

const ICONS = [Laptop, SquareTerminal, Link2] as const;

const summary = (checks: Check[]) => {
  if (checks.some((c) => c.state === "failed")) return "Needs a fix";
  const passed = checks.filter((c) => c.state === "passed").length;
  if (passed === checks.length) return "All passed";
  return `${passed} of ${checks.length} passed`;
};

interface ComputerChecksProps {
  computer: Computer;
}

// What Add a computer waits on, in the order it happens: the computer connects, its T3 Code is found, Nexul pairs it.
export const ComputerChecks = ({ computer }: ComputerChecksProps) => {
  const checks = computerChecks(computer);
  return (
    <div className="rounded-lg border border-border bg-card" role="status" aria-live="polite">
      <div className="flex items-center justify-between border-b border-border px-3 py-2">
        <span className="text-xs font-semibold">Checks</span>
        <span className="font-mono text-xs text-muted-foreground tabular-nums">{summary(checks)}</span>
      </div>
      <ul className="divide-y divide-border">
        {checks.map((check, i) => (
          <CheckRow key={check.name} icon={ICONS[i] ?? Laptop} check={check} />
        ))}
      </ul>
    </div>
  );
};
