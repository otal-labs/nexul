import { Play } from "lucide-react";

import { Button } from "@/components/ui/button";
import { SetupModelPicks } from "@/components/pairing/SetupModelPicks";
import { SetupPreselection } from "@/components/pairing/SetupPreselection";
import { SetupRunRows } from "@/components/pairing/SetupRunRows";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useFetchComputerSetup, useRunSetup } from "@/hooks/PairingHooks";
import { useSetupModels } from "@/hooks/useSetupModels";
import { setupRunRows, setupRunning, type Computer, type ComputerSetup } from "@/models/Pairing";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface SetupRunSectionProps {
  computerId: string;
  setup: ComputerSetup;
}

// Start, then the rows: the run just started lists its providers at once, a reopened dialog shows each provider's newest turn.
const SetupRunSection = ({ computerId, setup }: SetupRunSectionProps) => {
  const run = useRunSetup(computerId);
  const { choices, models, pick } = useSetupModels(computerId);
  const rows = setupRunRows(setup.turns, run.data);
  const busy = run.isPending || setupRunning(rows);
  const startLabel = setup.turns.length > 0 ? "Re-run setup" : "Start setup";
  return (
    <div className="space-y-4">
      {choices.length > 0 && <SetupModelPicks choices={choices} models={models} disabled={busy} onPick={pick} />}
      <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
        <Button type="button" onClick={() => run.mutate({ models })} disabled={busy}>
          <Play className="size-4" aria-hidden />
          {startLabel}
        </Button>
        {setup.confirmed_at && (
          <span className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <span className="size-1.5 rounded-full bg-success" aria-hidden />
            Confirmed {formatRelativeTime(setup.confirmed_at)}
          </span>
        )}
      </div>
      {rows.length > 0 && <SetupRunRows rows={rows} retryDisabled={busy} onRetry={(provider) => run.mutate({ models, provider })} />}
    </div>
  );
};

interface SetupStepProps {
  computer: Computer;
}

// Step three: one setup turn per provider connects Nexul's MCP server, installs the skills, and has the provider confirm itself.
export const SetupStep = ({ computer }: SetupStepProps) => {
  const { data: setup, error, isPending } = useFetchComputerSetup(computer.id);
  return (
    <div className="max-w-xl space-y-5">
      <p className="text-sm text-muted-foreground">
        Nexul runs one short setup turn per provider on {computer.name}. Each connects Nexul's MCP server with this
        computer's own token, installs the skills below, and confirms the skills its provider discovered.
      </p>
      <SetupPreselection />
      {isPending && <LoadingDisplay label="Reading setup" className="justify-start p-0" />}
      {error && <ErrorDisplay error={error} title="Couldn't read setup" className="p-4" />}
      {setup && <SetupRunSection computerId={computer.id} setup={setup} />}
    </div>
  );
};
