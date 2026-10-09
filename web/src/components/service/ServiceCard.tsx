import { Link } from "react-router";

import { DeployStatusBadge } from "@/components/service/DeployStatusBadge";
import { HealthDot } from "@/components/service/HealthDot";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import type { DeployStatus as DeployStatusType, ServiceDef } from "@/models/Service";

interface ServiceCardProps {
  service: ServiceDef;
  status?: DeployStatusType;
}

// Build repo for repo-driven services, else the machine it runs on; the list endpoint carries no deploy image.
const sourceLine = (service: ServiceDef): string => {
  const repo = service.build_source;
  if (repo?.repo_owner && repo?.repo_name) return `${repo.repo_owner}/${repo.repo_name}`;
  return service.machine || service.target;
};

// `status` degrades gracefully to no dot/badge until a per-service health query is wired up.
export const ServiceCard = ({ service, status }: ServiceCardProps) => {
  const canOpen = useAreaAccess()?.("stacks") ?? false;
  const wsPath = useWorkspacePath();
  const rowClass = "flex items-center gap-3 px-4 py-2.5";
  const source = sourceLine(service);
  const row = (
    <>
      {status && <HealthDot status={status} className="shrink-0" />}
      <span className="min-w-0 flex-1">
        <span className="block truncate text-sm font-medium">{service.name}</span>
        {source && <span className="block truncate font-mono text-xs text-muted-foreground">{source}</span>}
      </span>
      <span className="flex shrink-0 items-center gap-3">
        {status && <DeployStatusBadge status={status} />}
        <span className="font-mono text-xs text-muted-foreground">{service.strategy}</span>
      </span>
    </>
  );
  return (
    <li>
      {canOpen && (
        <Link
          to={wsPath(`/stacks/${service.id}`)}
          className={`${rowClass} transition-colors duration-[120ms] ease-standard hover:bg-accent/40`}
        >
          {row}
        </Link>
      )}
      {!canOpen && <div className={rowClass}>{row}</div>}
    </li>
  );
};
