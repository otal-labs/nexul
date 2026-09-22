import { Link } from "react-router";

import { HealthDot } from "@/components/service/HealthDot";
import { formatRelativeTime } from "@/components/service/DeployTime";
import { deployPath, type Deploy } from "@/models/Stack";

interface DeployRowProps {
  deploy: Deploy;
}

export const DeployRow = ({ deploy }: DeployRowProps) => (
  <li>
    <Link
      to={deployPath(deploy)}
      className="flex items-center gap-3 px-3 py-2.5 text-sm transition-colors duration-150 ease-standard hover:bg-accent/40"
    >
      <HealthDot status={deploy.status} className="shrink-0" />
      <span className="sr-only">{deploy.status}</span>
      <span className="min-w-0 flex-1 truncate font-mono text-xs" title={deploy.image || undefined}>
        {deploy.image || "repo build"}
      </span>
      {deploy.kind && <span className="hidden shrink-0 text-xs text-muted-foreground sm:inline">{deploy.kind}</span>}
      <span className="hidden max-w-32 shrink-0 truncate font-mono text-xs text-muted-foreground sm:inline" title={deploy.id}>
        {deploy.id}
      </span>
      <time dateTime={deploy.created_at} className="shrink-0 text-xs text-muted-foreground tabular-nums">
        {formatRelativeTime(deploy.created_at)}
      </time>
    </Link>
  </li>
);
