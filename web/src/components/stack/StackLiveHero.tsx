import type { ReactNode } from "react";
import { ArrowUpRightIcon } from "lucide-react";

import { HealthDot } from "@/components/service/HealthDot";
import { formatRelativeTime } from "@/components/service/DeployTime";
import { microheaderClass } from "@/components/Microheader";
import { useFetchStackServices } from "@/hooks/StackHooks";
import { cn } from "@/lib/utils";
import { ContainerStatus, type Container, type Deploy, type StackWithBranches } from "@/models/Stack";
import { errorRateHealth } from "@/utils/ErrorRateUtility";
import { formatClockTime } from "@/utils/TimeUtility";

interface StackLiveHeroProps {
  stack: StackWithBranches;
  latest: Deploy | undefined;
  hostnames: string[];
}

const down: string[] = [ContainerStatus.Exited, ContainerStatus.Stopped];

const Cell = ({ label, children }: { label: string; children: ReactNode }) => (
  <div className="min-w-0">
    <p className={microheaderClass}>{label}</p>
    <div className="mt-1.5 min-w-0">{children}</div>
  </div>
);

// A service is down while its outage is open, or when the runner last saw its container stopped.
const downNames = (services: Container[], stack: StackWithBranches): string[] => [
  ...new Set([...(stack.outages ?? []).map((o) => o.service), ...services.filter((c) => down.includes(c.status)).map((c) => c.name)]),
];

const Health = ({ services, stack }: { services: Container[] | undefined; stack: StackWithBranches }) => {
  const names = services ? downNames(services, stack) : [];
  const [only] = names;
  const label = names.length === 1 ? `${only} down` : `${names.length} services down`;
  const rate = errorRateHealth(stack);
  return (
    <Cell label="Health">
      {!services && <span className="text-sm text-muted-foreground">—</span>}
      {services && names.length === 0 && !rate && (
        <p className="flex items-center gap-2 text-sm font-medium">
          <span aria-hidden className="size-2 shrink-0 rounded-full bg-success" />
          No issues
        </p>
      )}
      {names.length > 0 && (
        <p className="flex min-w-0 items-center gap-2 text-sm font-medium">
          <span aria-hidden className="size-2 shrink-0 rounded-full bg-destructive" />
          <span className="min-w-0 truncate" title={names.join(", ")}>{label}</span>
        </p>
      )}
      {rate && (
        <>
          <p className="mt-0.5 flex min-w-0 items-center gap-2 text-sm font-medium first:mt-0">
            <span aria-hidden className={cn("size-2 shrink-0 rounded-full", rate.tone === "warning" ? "bg-warning" : "bg-success")} />
            <span className="min-w-0 truncate" title={rate.text}>{rate.text}</span>
          </p>
          <p className="mt-0.5 pl-4 font-mono text-xs text-muted-foreground">
            {rate.tone === "warning" ? "since" : "at"} {formatClockTime(rate.at)}
          </p>
        </>
      )}
    </Cell>
  );
};

const LastDeploy = ({ latest }: { latest: Deploy | undefined }) => (
  <Cell label="Last deploy">
    {latest && (
      <>
        <p className="flex items-center gap-2 text-sm font-medium capitalize">
          <HealthDot status={latest.status} />
          {latest.status}
        </p>
        <p className="mt-0.5 font-mono text-xs text-muted-foreground">{formatRelativeTime(latest.created_at)}</p>
      </>
    )}
    {!latest && <p className="text-sm text-muted-foreground">No deploys yet</p>}
  </Cell>
);

const Services = ({ services, stack }: { services: Container[] | undefined; stack: StackWithBranches }) => {
  const up = Math.max(0, (services?.length ?? 0) - (services ? downNames(services, stack).length : 0));
  return (
    <Cell label="Services">
      {!services && <span className="text-sm text-muted-foreground">—</span>}
      {services && (
        <>
          <p className="font-mono text-sm tabular-nums">{services.length}</p>
          <p className="mt-0.5 font-mono text-xs text-muted-foreground">{up} up</p>
        </>
      )}
    </Cell>
  );
};

const Hostnames = ({ hostnames }: { hostnames: string[] }) => {
  const [first, ...more] = hostnames;
  return (
    <Cell label="Hostnames">
      {first && (
        <a
          href={`https://${first}`}
          target="_blank"
          rel="noreferrer"
          className="group inline-flex max-w-full min-w-0 items-center gap-1 font-mono text-sm hover:underline hover:underline-offset-2"
        >
          <span className="truncate" title={first}>{first}</span>
          <ArrowUpRightIcon className="size-3 shrink-0 text-muted-foreground group-hover:text-foreground" aria-hidden />
        </a>
      )}
      {more.length > 0 && (
        <p className="mt-0.5 font-mono text-xs text-muted-foreground" title={more.join(", ")}>+{more.length} more</p>
      )}
      {!first && <p className="text-sm text-muted-foreground">None</p>}
    </Cell>
  );
};

// What a stack is doing right now, read before anything else on its page: one strip, health first.
export const StackLiveHero = ({ stack, latest, hostnames }: StackLiveHeroProps) => {
  const { data: services } = useFetchStackServices(stack.id);
  return (
    <div className="@container">
      <section
      aria-label="Live status"
      className={cn(
        "grid grid-cols-2 gap-x-6 gap-y-5 rounded-lg bg-surface-2 px-5 py-4 ring-1 ring-border",
        "@2xl:grid-cols-[minmax(0,1.2fr)_minmax(0,1fr)_minmax(0,0.7fr)_minmax(0,1.3fr)]",
      )}
    >
      <Health services={services} stack={stack} />
      <LastDeploy latest={latest} />
      <Services services={services} stack={stack} />
      <Hostnames hostnames={hostnames} />
      </section>
    </div>
  );
};
