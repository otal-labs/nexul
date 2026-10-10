import type { ReactNode } from "react";

import { ComputerMCPToken } from "@/components/settings/ComputerMCPToken";
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

interface FactProps {
  label: string;
  children: ReactNode;
}

const Fact = ({ label, children }: FactProps) => (
  <div className="grid gap-x-4 gap-y-1 @md:grid-cols-[7rem_minmax(0,1fr)]">
    <dt className="text-xs text-muted-foreground">{label}</dt>
    <dd className="min-w-0 text-xs">{children}</dd>
  </div>
);

interface ComputerDetailsProps {
  computer: Computer;
  setup: ComputerSetup | undefined;
}

// Everything a computer row folds away: where it is, what runs there, each provider's confirmation, and its MCP token.
export const ComputerDetails = ({ computer, setup }: ComputerDetailsProps) => {
  const lines = setup ? providerSetupLines(setup) : [];
  const pairing = stillPairing(computer);
  return (
    <dl className="@container space-y-3 rounded-md bg-surface-2 p-3">
      <Fact label="Address">
        <span className="font-mono break-all">{computer.server_url}</span>
      </Fact>
      {computer.harness_version && (
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
      <Fact label="MCP token">
        <ComputerMCPToken computerId={computer.id} />
      </Fact>
    </dl>
  );
};
