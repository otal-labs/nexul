import { ScrollTextIcon, XCircle } from "lucide-react";
import { Link } from "react-router";

import { NoFillBadge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { MonoRef } from "@/components/MonoRef";
import { ContainerStatusBadge } from "@/components/stack/ContainerStatusBadge";
import { ServiceOutageLine } from "@/components/stack/ServiceOutageLine";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { parseContainerPort, type Container, type ContainerPort, type ServiceOutage } from "@/models/Stack";
import { imageShortName } from "@/utils/ImageRefUtility";

interface ServiceRowProps {
  container: Container;
  hostnames: string[];
  outage?: ServiceOutage | undefined;
}

const imageOf = (c: Container): string => c.image || c.declared.image || c.declared.build || "";

const hostLabel = (host: string): string => (host.includes(":") ? host : `:${host}`);

const portText = (name: string, port: ContainerPort): string =>
  `${name}:${port.container}${port.proto !== "tcp" ? `/${port.proto}` : ""}${port.host ? ` → host ${hostLabel(port.host)}` : ""}`;

// How the service is reached: its first public hostname, else its first address on the stack's network; the rest in the tooltip.
const Reach = ({ name, hostnames, ports }: { name: string; hostnames: string[]; ports: ContainerPort[] }) => {
  const all = [...hostnames, ...ports.map((port) => portText(name, port))];
  const [host] = hostnames;
  const [port] = ports;
  return (
    <span className="flex min-w-0 items-center gap-1.5 font-mono text-xs" title={all.join("\n")}>
      {host && (
        <a
          href={`https://${host}`}
          target="_blank"
          rel="noreferrer"
          // As wide as the hostname, so "+N" follows it; narrower only when the column is.
          style={{ width: `${host.length}ch` }}
          className="min-w-0 shrink text-foreground underline-offset-2 hover:underline"
        >
          <MonoRef value={host} kind="hostname" />
        </a>
      )}
      {!host && port && <span className="truncate text-muted-foreground">{`${name}:${port.container}`}</span>}
      {all.length > 1 && <span className="shrink-0 text-muted-foreground">{`+${all.length - 1}`}</span>}
      {all.length === 0 && <span className="text-muted-foreground">no ports</span>}
    </span>
  );
};

const LogsButton = ({ container }: { container: Container }) => {
  const wsPath = useWorkspacePath();
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button asChild variant="ghost" size="icon" className="size-7 text-muted-foreground">
          <Link to={wsPath(`/stacks/${container.stack_id}/logs/${container.name}`)} aria-label={`Logs of ${container.name}`}>
            <ScrollTextIcon aria-hidden />
          </Link>
        </Button>
      </TooltipTrigger>
      <TooltipContent>Logs</TooltipContent>
    </Tooltip>
  );
};

// One line per service: name and status, the image, how it is reached, and its logs; an open outage adds why under it.
export const ServiceRow = ({ container, hostnames, outage }: ServiceRowProps) => {
  const canReadLogs = useAreaAccess()?.("stackLogs") ?? false;
  const image = imageOf(container);
  const ports = (container.ports ?? []).map(parseContainerPort).filter((p): p is ContainerPort => p != null);
  const addresses = [...hostnames, ...ports.map((port) => portText(container.name, port))];
  // Narrow, the reach column drops; the name's tooltip keeps the container and every address.
  const nameTitle = [container.container_name ? `container ${container.container_name}` : container.name, ...addresses].join("\n");
  // The 14px name centres 1px lower than the 12px mono beside it; lifted so the row shares one baseline.
  const name = (
    <span className="min-w-0 -translate-y-px truncate text-sm leading-4 font-medium text-foreground" title={nameTitle}>
      {container.name}
    </span>
  );
  const imageText = image && (
    <MonoRef value={imageShortName(image)} kind="image" title={image} className="text-xs text-muted-foreground" />
  );
  return (
    <li className="py-2.5">
      <div className="grid grid-cols-[minmax(0,1fr)_minmax(0,1.3fr)_6rem_1.75rem] items-center gap-x-6 @3xl:grid-cols-[minmax(0,1fr)_minmax(0,1.3fr)_minmax(0,1fr)_6rem_1.75rem]">
        {name}
        <span className="min-w-0">{imageText}</span>
        <span className="hidden min-w-0 @3xl:block">
          <Reach name={container.name} hostnames={hostnames} ports={ports} />
        </span>
        {/* An open outage outranks the runner's last word: the row reads down, and the line under it says why. */}
        <span className="justify-self-end">
          {outage && (
            <NoFillBadge icon={XCircle} color="text-destructive" className="text-foreground">
              down
            </NoFillBadge>
          )}
          {!outage && <ContainerStatusBadge status={container.status} />}
        </span>
        {canReadLogs && <LogsButton container={container} />}
      </div>
      {outage && (
        <div className="mt-1">
          <ServiceOutageLine outage={outage} />
        </div>
      )}
    </li>
  );
};
