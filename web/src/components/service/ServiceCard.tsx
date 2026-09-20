import { Link } from "react-router";

import { DeployStatusBadge } from "@/components/service/DeployStatusBadge";
import { HealthDot } from "@/components/service/HealthDot";
import type { DeployStatus as DeployStatusType, ServiceDef } from "@/models/Service";

// Stagger the first screenful of rows in; the rest mounts together instantly (never stagger a long list).
const STAGGER_LIMIT = 8;
const STAGGER_STEP_MS = 25;

interface ServiceCardProps {
  service: ServiceDef;
  status?: DeployStatusType;
  index?: number;
}

// Build repo for repo-driven services, target server otherwise; the list endpoint carries no deploy image.
const imageLine = (service: ServiceDef): string => {
  const repo = service.build_source;
  if (repo?.repo_owner && repo?.repo_name) return `${repo.repo_owner}/${repo.repo_name}`;
  return service.target;
};

// `status` degrades gracefully to no dot/badge until a per-service health query is wired up.
export const ServiceCard = ({ service, status, index }: ServiceCardProps) => (
  <li
    className={
      index !== undefined && index < STAGGER_LIMIT
        ? "animate-in fade-in-0 slide-in-from-bottom-1 fill-mode-backwards duration-150 ease-out"
        : undefined
    }
    style={index !== undefined && index < STAGGER_LIMIT ? { animationDelay: `${index * STAGGER_STEP_MS}ms` } : undefined}
  >
    <Link
      to={`/services/${service.id}`}
      className="flex items-center gap-3 px-4 py-3 transition-colors duration-[120ms] ease-standard hover:bg-accent/40"
    >
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
    </Link>
  </li>
);
