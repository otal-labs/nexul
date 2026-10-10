import { useState } from "react";

import { SetupFolderPick } from "@/components/pairing/SetupFolderPick";
import { SetupModelPicks } from "@/components/pairing/SetupModelPicks";
import { SetupOptions } from "@/components/pairing/SetupOptions";
import { SetupPreselection } from "@/components/pairing/SetupPreselection";
import { SetupRunHeader } from "@/components/pairing/SetupRunHeader";
import { SetupRunRows } from "@/components/pairing/SetupRunRows";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useFetchComputerSetup, useRunSetup, useUpdateSkills } from "@/hooks/ComputerSetupHooks";
import { useSetupChoices } from "@/hooks/useSetupChoices";
import { openSetupRow, setupRunRows, setupRunning, type Computer, type ComputerSetup } from "@/models/Pairing";

interface SetupRunSectionProps {
  computer: Computer;
  setup: ComputerSetup;
}

// The run on top, one row per provider under it, and the choices folded below once something has run.
const SetupRunSection = ({ computer, setup }: SetupRunSectionProps) => {
  const run = useRunSetup(computer.id);
  const update = useUpdateSkills(computer.id);
  const { choices, models, options, pick, pickOptions, included, excluded, include, projects, folder, pickFolder } = useSetupChoices(setup);
  // Undefined follows the run (the running row, else a failure); null is every row folded by hand.
  const [picked, setPicked] = useState<string | null>();
  const allRows = setupRunRows(setup.turns, run.data);
  const rows = allRows.filter((r) => !excluded.includes(r.provider));
  const open = picked === undefined ? openSetupRow(rows)?.provider : (picked ?? undefined);
  const busy = run.isPending || update.isPending || setupRunning(allRows);
  const noneIncluded = choices.length > 0 && included.length === 0;
  const folderTitle = projects.find((p) => p.path === folder)?.title;
  const summary = [choices.length > 0 && `${included.length} of ${choices.length} providers`, folderTitle].filter(Boolean).join(" · ");
  const start = (provider?: string) => {
    setPicked(undefined);
    run.mutate({ models, options, folder, providers: included, ...(provider ? { provider } : {}) });
  };
  // A failed skills update retries as an update; any other row re-runs that provider's setup.
  const retry = (provider: string) => {
    if (rows.find((r) => r.provider === provider)?.kind !== "skills") return start(provider);
    setPicked(undefined);
    update.mutate();
  };
  return (
    <div className="space-y-5">
      <SetupRunHeader setup={setup} rows={rows} busy={busy} blocked={noneIncluded} onStart={() => start()} />
      {noneIncluded && <p className="text-xs text-muted-foreground">Turn on at least one provider to start setup.</p>}
      {rows.length > 0 && (
        <SetupRunRows rows={rows} open={open} retryDisabled={busy} onToggle={(p) => setPicked(p === open ? null : p)} onRetry={retry} />
      )}
      <SetupOptions defaultOpen={setup.turns.length === 0} summary={summary}>
        <p className="text-sm text-muted-foreground">
          Each provider on {computer.name} takes one short turn to connect Nexul's MCP server with this computer's own token,
          install these skills, and confirm them.
        </p>
        {choices.length > 0 && (
          <SetupModelPicks
            choices={choices}
            models={models}
            options={options}
            included={included}
            disabled={busy}
            onPick={pick}
            onOptions={pickOptions}
            onInclude={include}
          />
        )}
        {projects.length > 0 && <SetupFolderPick projects={projects} folder={folder} disabled={busy} onPick={pickFolder} />}
        <SetupPreselection />
      </SetupOptions>
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
    <div>
      {isPending && <LoadingDisplay label="Reading setup" className="justify-start p-0" />}
      {error && <ErrorDisplay error={error} title="Couldn't read setup." className="p-4" />}
      {setup && <SetupRunSection computer={computer} setup={setup} />}
    </div>
  );
};
