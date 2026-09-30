import { useState, type ReactNode } from "react";
import { Play } from "lucide-react";

import { Button } from "@/components/ui/button";
import { SetupFolderPick } from "@/components/pairing/SetupFolderPick";
import { SetupModelPicks } from "@/components/pairing/SetupModelPicks";
import { SetupPreselection } from "@/components/pairing/SetupPreselection";
import { SetupRunRows } from "@/components/pairing/SetupRunRows";
import { SetupTranscript } from "@/components/pairing/SetupTranscript";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useFetchComputerSetup, useRunSetup } from "@/hooks/PairingHooks";
import { useSetupChoices } from "@/hooks/useSetupChoices";
import { followedSetupRow, setupRunRows, setupRunning, type Computer, type ComputerSetup } from "@/models/Pairing";
import { cn } from "@/lib/utils";
import { formatRelativeTime } from "@/utils/TimeUtility";

// From md up the controls and the provider list share one scrolling left column beside the transcript.
const CONTROLS = "space-y-6 md:p-6";
const LEFT_PANE = "md:min-h-0 md:w-[26rem] md:shrink-0 md:overflow-y-auto";

interface SetupLeadProps {
  name: string;
}

const SetupLead = ({ name }: SetupLeadProps) => (
  <>
    <p className="text-sm text-muted-foreground">
      Each provider on {name} takes one short turn to connect Nexul's MCP server with this computer's own token, install
      these skills, and confirm them.
    </p>
    <SetupPreselection />
  </>
);

interface SetupRunSectionProps {
  computerId: string;
  setup: ComputerSetup;
  children: ReactNode;
}

// The run just started lists its providers at once, a reopened dialog shows each provider's newest turn.
const SetupRunSection = ({ computerId, setup, children }: SetupRunSectionProps) => {
  const run = useRunSetup(computerId);
  const { choices, models, options, pick, pickOptions, included, excluded, include, projects, folder, pickFolder } = useSetupChoices(setup);
  // A clicked provider pins the transcript; Start and Retry hand it back to following the run.
  const [picked, setPicked] = useState<string>();
  const allRows = setupRunRows(setup.turns, run.data);
  const rows = allRows.filter((r) => !excluded.includes(r.provider));
  const selected = rows.find((r) => r.provider === picked) ?? followedSetupRow(rows);
  const running = rows.find((r) => r.state === "running");
  const busy = run.isPending || setupRunning(allRows);
  const noneIncluded = choices.length > 0 && included.length === 0;
  const startLabel = setup.turns.length > 0 ? "Re-run setup" : "Start setup";
  const start = (provider?: string) => {
    setPicked(undefined);
    run.mutate({ models, options, folder, providers: included, ...(provider ? { provider } : {}) });
  };
  return (
    <>
      <div className={LEFT_PANE}>
        <div className={CONTROLS}>
          {children}
          {projects.length > 0 && <SetupFolderPick projects={projects} folder={folder} disabled={busy} onPick={pickFolder} />}
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
          <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
            <Button type="button" onClick={() => start()} loading={busy} disabled={noneIncluded}>
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
          {noneIncluded && <p className="text-xs text-muted-foreground">Turn on at least one provider to start setup.</p>}
        </div>
        {rows.length > 0 && (
          <div className="max-md:mt-6 md:px-6 md:pb-6">
            <SetupRunRows rows={rows} selected={selected?.provider} retryDisabled={busy} onSelect={setPicked} onRetry={start} />
          </div>
        )}
      </div>
      <SetupTranscript row={selected} runningName={running?.name} retryDisabled={busy} onRetry={start} />
    </>
  );
};

interface SetupStepProps {
  computer: Computer;
}

// Step three: one setup turn per provider connects Nexul's MCP server, installs the skills, and has the provider confirm itself.
export const SetupStep = ({ computer }: SetupStepProps) => {
  const { data: setup, error, isPending } = useFetchComputerSetup(computer.id);
  return (
    <div className="flex flex-col md:h-full md:flex-row">
      {!setup && (
        <div className={cn(LEFT_PANE, CONTROLS)}>
          <SetupLead name={computer.name} />
          {isPending && <LoadingDisplay label="Reading setup" className="justify-start p-0" />}
          {error && <ErrorDisplay error={error} title="Couldn't read setup" className="p-4" />}
        </div>
      )}
      {setup && (
        <SetupRunSection computerId={computer.id} setup={setup}>
          <SetupLead name={computer.name} />
        </SetupRunSection>
      )}
    </div>
  );
};
