import { Link } from "react-router";

import { HealthDot } from "@/components/service/HealthDot";
import { formatRelativeTime } from "@/components/service/DeployTime";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { deployPath, type Deploy } from "@/models/Stack";

interface DeployRowProps {
  deploy: Deploy;
}

export const DeployRow = ({ deploy }: DeployRowProps) => {
  const canOpen = useAreaAccess()?.("deploys") ?? false;
  const wsPath = useWorkspacePath();
  const rowClass = "flex items-center gap-3 px-3 py-2.5 text-sm";
  const row = (
    <>
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
    </>
  );

  return (
    <li>
      {canOpen && (
        <Link
          to={wsPath(deployPath(deploy))}
          className={`${rowClass} transition-colors duration-150 ease-standard hover:bg-accent/40`}
        >
          {row}
        </Link>
      )}
      {!canOpen && <div className={rowClass}>{row}</div>}
    </li>
  );
};
