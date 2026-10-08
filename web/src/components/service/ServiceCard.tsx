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

// Build repo for repo-driven services, target server otherwise; the list endpoint carries no deploy image.
const imageLine = (service: ServiceDef): string => {
  const repo = service.build_source;
  if (repo?.repo_owner && repo?.repo_name) return `${repo.repo_owner}/${repo.repo_name}`;
  return service.target;
};

// `status` degrades gracefully to no dot/badge until a per-service health query is wired up.
export const ServiceCard = ({ service, status }: ServiceCardProps) => {
  const canOpen = useAreaAccess()?.("stacks") ?? false;
  const wsPath = useWorkspacePath();
  const rowClass = "flex items-center gap-3 px-4 py-3";
  const row = (
    <>
      {status && <HealthDot status={status} className="shrink-0" />}
      <span className="min-w-0 flex-1 truncate font-medium">{service.name}</span>
      <span className="hidden w-48 shrink-0 truncate font-mono text-xs text-muted-foreground sm:inline">
        {imageLine(service)}
      </span>
      <span className="flex shrink-0 items-center gap-3">
        {status && <DeployStatusBadge status={status} />}
        <span className="hidden w-20 shrink-0 text-right font-mono text-xs text-muted-foreground sm:inline">
          {service.strategy}
        </span>
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
