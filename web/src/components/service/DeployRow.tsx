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
  const rowClass = "flex items-start gap-3 px-3 py-2.5 text-sm";
  const row = (
    <>
      <HealthDot status={deploy.status} className="mt-1 shrink-0" />
      <span className="sr-only">{deploy.status}</span>
      <div className="min-w-0 flex-1 space-y-0.5 font-mono text-xs">
        <p className="wrap-anywhere">{deploy.image || "repo build"}</p>
        <p className="flex flex-wrap gap-x-2 text-muted-foreground">
          {deploy.kind && <span>{deploy.kind}</span>}
          <span className="wrap-anywhere">{deploy.id}</span>
        </p>
      </div>
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
