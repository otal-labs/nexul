import type { ReactNode } from "react";
import { ArrowUpRightIcon } from "lucide-react";

import { HealthDot } from "@/components/service/HealthDot";
import { formatRelativeTime } from "@/components/service/DeployTime";
import { microheaderClass } from "@/components/Microheader";
import { useFetchStackServices } from "@/hooks/StackHooks";
import { cn } from "@/lib/utils";
import { ContainerStatus, type Container, type Deploy } from "@/models/Stack";

interface StackLiveHeroProps {
  stackId: string;
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

const Health = ({ services }: { services: Container[] | undefined }) => {
  const stopped = (services ?? []).filter((c) => down.includes(c.status));
  const [only] = stopped;
  const label = stopped.length === 1 ? `${only?.name} not running` : `${stopped.length} not running`;
  return (
    <Cell label="Health">
      {!services && <span className="text-sm text-muted-foreground">—</span>}
      {services && stopped.length === 0 && (
        <p className="flex items-center gap-2 text-sm font-medium">
          <span aria-hidden className="size-2 shrink-0 rounded-full bg-success" />
          No issues
        </p>
      )}
      {stopped.length > 0 && (
        <p className="flex min-w-0 items-center gap-2 text-sm font-medium">
          <span aria-hidden className="size-2 shrink-0 rounded-full bg-destructive" />
          <span className="min-w-0 truncate" title={label}>{label}</span>
        </p>
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

const Services = ({ services }: { services: Container[] | undefined }) => {
  const up = (services ?? []).filter((c) => !down.includes(c.status)).length;
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
export const StackLiveHero = ({ stackId, latest, hostnames }: StackLiveHeroProps) => {
  const { data: services } = useFetchStackServices(stackId);
  return (
    <div className="@container">
      <section
      aria-label="Live status"
      className={cn(
        "grid grid-cols-2 gap-x-6 gap-y-5 rounded-lg bg-surface-2 px-5 py-4 ring-1 ring-border",
        "@2xl:grid-cols-[minmax(0,1.2fr)_minmax(0,1fr)_minmax(0,0.7fr)_minmax(0,1.3fr)]",
      )}
    >
      <Health services={services} />
      <LastDeploy latest={latest} />
      <Services services={services} />
      <Hostnames hostnames={hostnames} />
      </section>
    </div>
  );
};
