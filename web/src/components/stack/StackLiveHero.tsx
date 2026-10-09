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

const containerDot: Record<Container["status"], string> = {
  [ContainerStatus.Pending]: "bg-warning",
  [ContainerStatus.Running]: "bg-info",
  [ContainerStatus.Healthy]: "bg-success",
  [ContainerStatus.Exited]: "bg-destructive",
  [ContainerStatus.Stopped]: "bg-muted-foreground",
};

const down: string[] = [ContainerStatus.Exited, ContainerStatus.Stopped];
const MAX_SERVICES = 4;

const ServiceLine = ({ service }: { service: Container }) => (
  <li className="flex min-w-0 items-center gap-2">
    <span aria-hidden className={cn("size-1.5 shrink-0 rounded-full", containerDot[service.status])} />
    <span className="min-w-0 truncate font-mono text-xs" title={service.name}>{service.name}</span>
    <span className="ml-auto shrink-0 text-xs text-muted-foreground">{service.status}</span>
  </li>
);

// What a stack is doing right now, read before anything else on its page: the last deploy, its services, where it is reached.
export const StackLiveHero = ({ stackId, latest, hostnames }: StackLiveHeroProps) => {
  const { data: services } = useFetchStackServices(stackId);
  const stopped = (services ?? []).filter((c) => down.includes(c.status)).length;
  const up = (services?.length ?? 0) - stopped;
  const [firstHostname, ...moreHostnames] = hostnames;

  const state = (
    <div className="min-w-0">
      <p className={microheaderClass}>Last deploy</p>
      {latest && (
        <>
          <p className={"mt-1.5 flex items-center gap-2.5 text-xl font-semibold capitalize"}>
            <HealthDot status={latest.status} className="size-2.5" />
            {latest.status}
          </p>
          <p className="mt-1 font-mono text-xs text-muted-foreground">deployed {formatRelativeTime(latest.created_at)}</p>
        </>
      )}
      {!latest && <p className="mt-1.5 text-sm text-muted-foreground">No deploys yet</p>}
    </div>
  );

  const servicesBlock = (
    <div className="min-w-0">
      <p className={microheaderClass}>Services</p>
      <p className="mt-1.5 flex flex-wrap items-baseline gap-x-2">
        {!services && <span className="text-muted-foreground">—</span>}
        {services && (
          <span className={"font-mono text-sm tabular-nums"}>
            {services.length === 1 ? "1 service" : `${services.length} services`}
          </span>
        )}
        {stopped > 0 && <span className="text-xs text-destructive">{stopped} not running</span>}
        {services && services.length > 0 && stopped === 0 && <span className="text-xs text-muted-foreground">{up} up</span>}
      </p>
      {services && services.length > 0 && (
        <ul className="mt-2.5 space-y-1.5">
          {services.slice(0, MAX_SERVICES).map((s) => <ServiceLine key={s.id} service={s} />)}
          {services.length > MAX_SERVICES && (
            <li className="font-mono text-xs text-muted-foreground">+{services.length - MAX_SERVICES} more</li>
          )}
        </ul>
      )}
    </div>
  );

  const reach = (
    <div className="min-w-0">
      <p className={microheaderClass}>Hostnames</p>
      <div className="mt-1.5 flex min-w-0 flex-col gap-1">
        {firstHostname && (
          <a
            href={`https://${firstHostname}`}
            target="_blank"
            rel="noreferrer"
            className="group inline-flex min-w-0 items-center gap-1 font-mono text-xs wrap-anywhere hover:underline hover:underline-offset-2"
          >
            {firstHostname}
            <ArrowUpRightIcon className="size-3 shrink-0 text-muted-foreground group-hover:text-foreground" aria-hidden />
          </a>
        )}
        {moreHostnames.length > 0 && (
          <span className="font-mono text-xs text-muted-foreground" title={moreHostnames.join(", ")}>
            +{moreHostnames.length} more
          </span>
        )}
        {!firstHostname && <span className="text-sm text-muted-foreground">None</span>}
      </div>
    </div>
  );

  return (
    <section
      aria-label="Live status"
      className="grid gap-y-5 rounded-lg bg-surface-2 p-5 ring-1 ring-border @xl:grid-cols-[minmax(0,1fr)_minmax(0,1.2fr)_minmax(0,1fr)] @xl:divide-x @xl:divide-border @xl:*:px-6 @xl:*:first:pl-0 @xl:*:last:pr-0"
    >
      {state}
      {servicesBlock}
      {reach}
    </section>
  );
};
