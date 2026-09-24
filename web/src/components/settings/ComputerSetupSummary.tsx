import { Wrench } from "lucide-react";

import { Button } from "@/components/ui/button";
import { PairComputerDialog } from "@/components/pairing/PairComputerDialog";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useFetchComputerSetup } from "@/hooks/PairingHooks";
import { providerSetupLines, type Computer, type ComputerSetup, type ProviderSetupLine } from "@/models/Pairing";
import { cn } from "@/lib/utils";
import { formatRelativeTime } from "@/utils/TimeUtility";

const LINE_DOT: Record<ProviderSetupLine["state"], string> = {
  confirmed: "bg-success",
  running: "bg-warning animate-[status-pulse_2.4s_ease-standard_infinite]",
  failed: "bg-destructive",
  unconfirmed: "bg-muted-foreground/40",
};

const lineLabel = (line: ProviderSetupLine) => {
  if (line.state === "running") return "setting up…";
  if (line.state === "confirmed" && line.confirmedAt) return `confirmed ${formatRelativeTime(line.confirmedAt)}`;
  if (line.state === "failed") return "setup failed";
  return "not confirmed";
};

interface ProviderLineProps {
  line: ProviderSetupLine;
}

const ProviderLine = ({ line }: ProviderLineProps) => (
  <li className="flex min-w-0 items-center gap-1.5 text-xs text-muted-foreground">
    <span className={cn("size-1.5 shrink-0 rounded-full transition-colors duration-150 ease-standard", LINE_DOT[line.state])} aria-hidden />
    <span className="break-words text-foreground">{line.name}</span>
    <span className="font-mono tabular-nums">{lineLabel(line)}</span>
  </li>
);

interface SetupDetailsProps {
  computer: Computer;
  setup: ComputerSetup;
}

// Read-only by design: the button only opens the dialog, and an agent alone changes a confirmation through MCP.
const SetupDetails = ({ computer, setup }: SetupDetailsProps) => {
  const lines = providerSetupLines(setup);
  const confirmed = setup.confirmed_at !== null;
  return (
    <div className="flex flex-wrap items-start justify-between gap-x-3 gap-y-2">
      <div className="min-w-0 space-y-1">
        <p className="flex items-center gap-1.5 text-xs font-medium">
          <span className={cn("size-1.5 shrink-0 rounded-full", confirmed ? "bg-success" : "bg-warning")} aria-hidden />
          {confirmed ? "Setup confirmed" : "Needs setup"}
        </p>
        {lines.length > 0 && (
          <ul className="space-y-0.5 pl-3">
            {lines.map((line) => (
              <ProviderLine key={line.provider} line={line} />
            ))}
          </ul>
        )}
      </div>
      <PairComputerDialog
        setupFor={computer}
        trigger={
          <Button type="button" variant="outline" size="sm">
            <Wrench className="size-4" aria-hidden />
            {confirmed ? "Re-run setup" : "Set up"}
          </Button>
        }
      />
    </div>
  );
};

interface ComputerSetupSummaryProps {
  computer: Computer;
}

// The setup badge, one line per provider, and the way into the Set up step; setup pushes keep it live.
export const ComputerSetupSummary = ({ computer }: ComputerSetupSummaryProps) => {
  const { data: setup, error, isPending } = useFetchComputerSetup(computer.id);
  return (
    <div className="border-t border-border pt-2">
      {isPending && <LoadingDisplay label="Reading setup" className="justify-start p-0" />}
      {error && <ErrorDisplay error={error} title="Couldn't read setup" className="p-3" />}
      {setup && <SetupDetails computer={computer} setup={setup} />}
    </div>
  );
};
