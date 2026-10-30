import { useState } from "react";

import { EnterList } from "@/components/EnterList";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { SetupFolderPick } from "@/components/pairing/SetupFolderPick";
import { SetupProviderItem } from "@/components/pairing/SetupProviderItem";
import { SetupRunHeader } from "@/components/pairing/SetupRunHeader";
import { useFetchComputerSetup, useRunSetup, useUpdateSkills } from "@/hooks/ComputerSetupHooks";
import { useFetchHarnessProviders } from "@/hooks/PairingHooks";
import { useSetupChoices } from "@/hooks/useSetupChoices";
import { openSetupRow, setupProviderRows, setupRunRows, type Computer, type ComputerSetup } from "@/models/Pairing";
import { cn } from "@/lib/utils";

// The right padding gives the kept scrollbar gutter's 10px back, so the rows end on the run block's edge. The line under the run
// block is the first row's own top edge, so a row lifted to the top under the 12px scroll padding shows one hairline, not two.
const SCROLL_FRAME =
  "scroll-edge -mx-4 min-h-0 flex-1 overflow-y-auto pr-1.5 pl-4 [scroll-padding-block:12px_40px] [scrollbar-gutter:stable] sm:-mx-6 sm:pr-3.5 sm:pl-6";

interface SetupRunSectionProps {
  setup: ComputerSetup;
  framed: boolean;
}

// The run on top, then one row per provider with its own switch, model and run; the providers run side by side.
const SetupRunSection = ({ setup, framed }: SetupRunSectionProps) => {
  const run = useRunSetup(setup.computer_id);
  const update = useUpdateSkills(setup.computer_id);
  const { choices, models, options, pick, pickOptions, included, include, projects, folder, pickFolder } = useSetupChoices(setup);
  const listing = useFetchHarnessProviders(setup.computer_id);
  const providers = listing.isPending ? "pending" : listing.error ? "error" : "ready";
  // Undefined follows the run (a running row, else a failure); null is every row folded by hand.
  const [picked, setPicked] = useState<string | null>();
  const rows = setupProviderRows(choices, setupRunRows(setup.turns, run.data), setup, providers === "ready");
  const on = rows.filter((r) => r.on);
  const counted = on.filter((r) => r.choice);
  const running = on.some((r) => r.state === "running" || r.state === "queued");
  const open = picked === undefined ? openSetupRow(on.flatMap((r) => (r.run ? [r.run] : [])))?.provider : (picked ?? undefined);
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
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="space-y-4 pb-4">
        <SetupRunHeader
          rows={counted}
          providers={providers}
          busy={pending === "*"}
          running={running}
          confirmed={!!setup.confirmed_at}
          onStart={startAll}
          onRetry={() => void listing.refetch()}
          retrying={listing.isFetching}
        />
        <SetupFolderPick projects={projects} folder={folder} disabled={running} onPick={pickFolder} />
      </div>
      {rows.length > 0 && (
        <EnterList aria-label="Providers" className={cn("divide-y divide-border *:first:border-t", framed && SCROLL_FRAME)}>
          {rows.map((row) => (
            <SetupProviderItem
              key={row.provider}
              row={row}
              model={models[row.provider] ?? ""}
              options={options[row.provider] ?? []}
              open={row.provider === open}
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
    </div>
  );
};

interface SetupStepProps {
  computer: Computer;
  // Inside a dialog's fixed frame the run block stays put and only the list scrolls, out to the frame's edges.
  framed?: boolean;
}

// Step three: one setup turn per switched-on provider connects Nexul's MCP server, installs the skills, and has the provider confirm itself.
export const SetupStep = ({ computer, framed = false }: SetupStepProps) => {
  const { data: setup, error, isPending } = useFetchComputerSetup(computer.id);
  return (
    <div className="flex min-h-0 flex-1 flex-col">
      {isPending && <LoadingDisplay label="Reading setup" className="justify-start p-0" />}
      {error && <ErrorDisplay error={error} title="Couldn't read setup." className="p-4" />}
      {setup && <SetupRunSection setup={setup} framed={framed} />}
    </div>
  );
};
