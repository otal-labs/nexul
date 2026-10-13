import { useState } from "react";

import { EnterList } from "@/components/EnterList";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { SetupFolderPick } from "@/components/pairing/SetupFolderPick";
import { SetupProviderItem } from "@/components/pairing/SetupProviderItem";
import { SetupRunHeader } from "@/components/pairing/SetupRunHeader";
import { useFetchComputerSetup, useRunSetup, useUpdateSkills } from "@/hooks/ComputerSetupHooks";
import { useSetupChoices } from "@/hooks/useSetupChoices";
import { needsSetup, openSetupRow, setupProviderRows, setupRunRows, type Computer, type ComputerSetup } from "@/models/Pairing";

interface SetupRunSectionProps {
  setup: ComputerSetup;
}

// The run on top, then one row per provider with its own switch, model and run; the providers run side by side.
const SetupRunSection = ({ setup }: SetupRunSectionProps) => {
  const run = useRunSetup(setup.computer_id);
  const update = useUpdateSkills(setup.computer_id);
  const { choices, models, options, pick, pickOptions, included, include, projects, folder, pickFolder } = useSetupChoices(setup);
  // Undefined follows the run (a running row, else a failure); null is every row folded by hand.
  const [picked, setPicked] = useState<string | null>();
  const rows = setupProviderRows(choices, setupRunRows(setup.turns, run.data), setup);
  const on = rows.filter((r) => r.on);
  const running = on.some((r) => r.state === "running" || r.state === "queued");
  const open = picked === undefined ? openSetupRow(on.flatMap((r) => (r.run ? [r.run] : [])))?.provider : (picked ?? undefined);
  const next = on.find(needsSetup)?.provider;
  // The one request in flight: a single provider's run, or "*" for Set up all and a skills update.
  const pending = (run.isPending && (run.variables?.provider ?? "*")) || (update.isPending && "*") || undefined;

  const startAll = () => {
    setPicked(undefined);
    run.mutate({ models, options, folder, providers: included });
  };
  // A failed skills update retries as an update; any other row runs that provider's setup on its own.
  const runOne = (provider: string) => {
    setPicked(undefined);
    if (rows.find((r) => r.provider === provider)?.run?.kind === "skills") return update.mutate();
    run.mutate({ models, options, folder, provider });
  };
  const toggle = (provider: string) => setPicked(provider === open ? null : provider);

  return (
    <div className="space-y-5">
      <SetupRunHeader rows={on} busy={pending === "*"} running={running} confirmed={!!setup.confirmed_at} onStart={startAll} />
      {rows.length > 0 && (
        <EnterList aria-label="Providers" className="@container divide-y divide-border overflow-hidden rounded-lg border border-border bg-card">
          {rows.map((row) => (
            <SetupProviderItem
              key={row.provider}
              row={row}
              model={models[row.provider] ?? ""}
              options={options[row.provider] ?? []}
              open={row.provider === open}
              next={row.provider === next}
              busy={pending === row.provider}
              onInclude={include}
              onPick={pick}
              onOptions={pickOptions}
              onRun={runOne}
              onToggle={toggle}
            />
          ))}
        </EnterList>
      )}
      <SetupFolderPick projects={projects} folder={folder} disabled={running} onPick={pickFolder} />
    </div>
  );
};

interface SetupStepProps {
  computer: Computer;
}

// Step three: one setup turn per switched-on provider connects Nexul's MCP server, installs the skills, and has the provider confirm itself.
export const SetupStep = ({ computer }: SetupStepProps) => {
  const { data: setup, error, isPending } = useFetchComputerSetup(computer.id);
  return (
    <div>
      {isPending && <LoadingDisplay label="Reading setup" className="justify-start p-0" />}
      {error && <ErrorDisplay error={error} title="Couldn't read setup." className="p-4" />}
      {setup && <SetupRunSection setup={setup} />}
    </div>
  );
};
