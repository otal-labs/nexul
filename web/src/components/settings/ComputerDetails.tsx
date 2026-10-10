import { ComputerFact as Fact } from "@/components/settings/ComputerFact";
import { ComputerFactRows } from "@/components/settings/ComputerFactRows";
import { ComputerMCPToken } from "@/components/settings/ComputerMCPToken";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { reachedThroughRunner, t3Status } from "@/models/ComputerFacts";
import {
  harnessLabel,
  providerSetupLines,
  SKILLS_OUTDATED,
  stillPairing,
  type Computer,
  type ComputerSetup,
  type ProviderSetupLine,
} from "@/models/Pairing";
import { cn } from "@/lib/utils";
import { formatRelativeTime, formatShortDate } from "@/utils/TimeUtility";

const LINE_DOT: Record<ProviderSetupLine["state"], string> = {
  confirmed: "bg-success",
  running: "bg-warning animate-[status-pulse_2.4s_ease-standard_infinite]",
  failed: "bg-destructive",
  unconfirmed: "bg-muted-foreground/40",
};

const lineLabel = (line: ProviderSetupLine) => {
  if (line.state === "running" && line.kind === "skills") return "updating skills…";
  if (line.state === "running") return "setting up…";
  if (line.state === "confirmed" && line.skillsOutdated) return SKILLS_OUTDATED;
  if (line.state === "confirmed" && line.confirmedAt) return `confirmed ${formatRelativeTime(line.confirmedAt)}`;
  if (line.state === "failed") return "setup failed";
  return "not confirmed";
};

interface ProviderLineProps {
  line: ProviderSetupLine;
}

const ProviderLine = ({ line }: ProviderLineProps) => (
  <li className="flex min-w-0 items-center gap-1.5 text-xs text-muted-foreground">
    <span
      className={cn(
        "size-1.5 shrink-0 rounded-full transition-colors duration-150 ease-standard",
        line.state === "confirmed" && line.skillsOutdated ? "bg-info" : LINE_DOT[line.state],
      )}
      aria-hidden
    />
    <span className="break-words text-foreground">{line.name}</span>
    <span className="font-mono tabular-nums">{lineLabel(line)}</span>
  </li>
);

interface ComputerDetailsProps {
  computer: Computer;
  setup: ComputerSetup | undefined;
}

// Everything a computer row folds away: where it is, what runs there, each provider's confirmation, its facts, and its
// MCP token. A computer reached through its runner shows its T3 Code instead of the relay address.
export const ComputerDetails = ({ computer, setup }: ComputerDetailsProps) => {
  const lines = setup ? providerSetupLines(setup) : [];
  const pairing = stillPairing(computer);
  const t3 = computer.facts && t3Status(computer.facts.t3);
  const { restart_error: restartError, restarted_at: restartedAt } = computer.facts?.t3 ?? {};
  return (
    <dl className="@container space-y-3 rounded-md bg-surface-2 p-3">
      {computer.server_url && !reachedThroughRunner(computer.server_url) && (
        <Fact label="Address">
          <span className="font-mono break-all">{computer.server_url}</span>
        </Fact>
      )}
      {t3 && (
        <Fact label="T3 Code">
          <span className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-0.5">
            <SettingsStatus tone={t3.tone}>{t3.text}</SettingsStatus>
            {t3.detail && <span className="min-w-0 font-mono break-all text-muted-foreground tabular-nums">{t3.detail}</span>}
          </span>
          {restartError && <span className="mt-1 block break-words text-destructive">Restart failed: {restartError}</span>}
          {!restartError && restartedAt && (
            <span className="mt-1 block font-mono text-muted-foreground tabular-nums">restarted {formatRelativeTime(restartedAt)}</span>
          )}
        </Fact>
      )}
      {!t3 && computer.harness_version && (
        <Fact label={harnessLabel(computer.kind)}>
          <span className="font-mono">{computer.harness_version}</span>
        </Fact>
      )}
      {!pairing && (
        <Fact label="Paired until">
          <span className="font-mono">{formatShortDate(computer.token_expires_at)}</span>
        </Fact>
      )}
      {lines.length > 0 && (
        <Fact label="Providers">
          <ul className="space-y-1">
            {lines.map((line) => (
              <ProviderLine key={line.provider} line={line} />
            ))}
          </ul>
        </Fact>
      )}
      {computer.facts && <ComputerFactRows facts={computer.facts} factsAt={computer.facts_at} />}
      <Fact label="MCP token">
        <ComputerMCPToken computerId={computer.id} />
      </Fact>
    </dl>
  );
};
