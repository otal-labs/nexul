import { ScrollTextIcon } from "lucide-react";
import { Link } from "react-router";

import { ContainerStatusBadge } from "@/components/stack/ContainerStatusBadge";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { parseContainerPort, type Container, type ContainerPort } from "@/models/Stack";

interface ServiceRowProps {
  container: Container;
  hostnames: string[];
}

const imageOf = (c: Container): string => c.image || c.declared.image || c.declared.build || "";

const hostLabel = (host: string): string => (host.includes(":") ? host : `:${host}`);

const PortItem = ({ name, port }: { name: string; port: ContainerPort }) => (
  <>
    <li className="text-muted-foreground" title="Address on the stack's network">
      {name}:{port.container}
      {port.proto !== "tcp" && `/${port.proto}`}
    </li>
    {port.host && (
      <li className="text-muted-foreground/70" title="Published on the machine">
        host {hostLabel(port.host)}
      </li>
    )}
  </>
);

// Trailing column reads how the service is reached: public hostnames first, then its address on the stack's network.
export const ServiceRow = ({ container, hostnames }: ServiceRowProps) => {
  const wsPath = useWorkspacePath();
  const canReadLogs = useAreaAccess()?.("stackLogs") ?? false;
  const image = imageOf(container);
  const ports = (container.ports ?? []).map(parseContainerPort).filter((p): p is ContainerPort => p != null);

  return (
    <li className="grid grid-cols-[minmax(0,1fr)_auto] items-start gap-x-8 px-4 py-3">
      <div className="min-w-0 space-y-1">
        <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
          <span className="text-sm font-medium text-foreground">{container.name}</span>
          <ContainerStatusBadge status={container.status} />
          {canReadLogs && (
            <Link
              to={wsPath(`/stacks/${container.stack_id}/logs/${container.name}`)}
              className="inline-flex items-center gap-1 font-mono text-xs text-muted-foreground underline-offset-2 transition-colors duration-150 hover:text-foreground hover:underline"
            >
              <ScrollTextIcon className="size-3.5" aria-hidden /> Logs
            </Link>
          )}
        </div>
        {image && (
          <p className="truncate font-mono text-xs text-muted-foreground" title={image}>
            {image}
          </p>
        )}
        {container.container_name && (
          <p className="truncate font-mono text-[11px] text-muted-foreground/70" title="Docker container name">
            container {container.container_name}
          </p>
        )}
      </div>
      <ul className="max-w-72 space-y-1 text-right font-mono text-xs" aria-label={`Addresses of ${container.name}`}>
        {hostnames.map((hostname) => (
          <li key={hostname}>
            <a
              href={`https://${hostname}`}
              target="_blank"
              rel="noreferrer"
              className="break-all text-foreground underline underline-offset-2"
            >
              {hostname}
            </a>
          </li>
        ))}
        {ports.map((port) => (
          <PortItem key={`${port.host ?? ""}:${port.container}/${port.proto}`} name={container.name} port={port} />
        ))}
        {hostnames.length === 0 && ports.length === 0 && <li className="text-muted-foreground">no ports</li>}
      </ul>
    </li>
  );
};
