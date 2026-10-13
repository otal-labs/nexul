import { Server } from "lucide-react";

import { CheckRow } from "@/components/pairing/CheckRow";
import { useRunners } from "@/hooks/RunnerHooks";
import type { Check } from "@/models/ComputerChecks";
import type { Runner } from "@/models/Runner";

const serverCheck = (runner: Runner | undefined): Check => {
  const base = { name: "Server connected" };
  if (!runner) return { ...base, detail: "Run the command on the machine", state: "waiting", label: "Waiting" };
  return { ...base, detail: [runner.name, runner.machine].filter(Boolean).join(" · "), state: "passed", label: "Connected" };
};

interface RunnerConnectedCheckProps {
  name: string;
}

// Ticks when a runner under this name connects; the runners list refetches on each connect frame.
export const RunnerConnectedCheck = ({ name }: RunnerConnectedCheckProps) => {
  const { data: runners } = useRunners();
  const check = serverCheck(runners?.find((r) => r.name === name && r.connected));

  return (
    <div className="rounded-lg border border-border bg-card" role="status" aria-live="polite">
      <ul>
        <CheckRow icon={Server} check={check} />
      </ul>
    </div>
  );
};
