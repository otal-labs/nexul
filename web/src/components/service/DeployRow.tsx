import { Link } from "react-router";

import { DeployStatusBadge } from "@/components/service/DeployStatusBadge";
import { formatRelativeTime } from "@/components/service/DeployTime";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { deployPath, type Deploy } from "@/models/Stack";

interface DeployRowProps {
  deploy: Deploy;
}

// Status leads in a fixed column so the image after it never shifts between healthy and failed.
export const DeployRow = ({ deploy }: DeployRowProps) => {
  const canOpen = useAreaAccess()?.("deploys") ?? false;
  const wsPath = useWorkspacePath();
  const rowClass = "grid grid-cols-[5.5rem_minmax(0,1fr)_auto] items-start gap-3 px-3 py-2.5 text-sm";
  const facts = [
    deploy.kind && deploy.kind !== "deploy" ? deploy.kind : undefined,
    deploy.triggered_by && `by ${deploy.triggered_by}`,
    deploy.rule_name && `rule ${deploy.rule_name}`,
    deploy.pr_number && `#${deploy.pr_number}`,
  ].filter(Boolean);
  const row = (
    <>
      <span className="pt-px">
        <DeployStatusBadge status={deploy.status} />
      </span>
      <div className="min-w-0 space-y-0.5 font-mono text-xs">
        <p className="wrap-anywhere">{deploy.image || "repo build"}</p>
        <p className="flex flex-wrap gap-x-2 text-muted-foreground">
          {facts.map((fact) => (
            <span key={fact}>{fact}</span>
          ))}
          <span title={deploy.id}>{deploy.id.slice(0, 8)}</span>
        </p>
      </div>
      <time dateTime={deploy.created_at} className="pt-px font-mono text-xs text-muted-foreground tabular-nums">
        {formatRelativeTime(deploy.created_at)}
      </time>
    </>
  );

  return (
    <li>
      {canOpen && (
        <Link to={wsPath(deployPath(deploy))} className={`${rowClass} transition-colors duration-150 ease-standard hover:bg-accent/40`}>
          {row}
        </Link>
      )}
      {!canOpen && <div className={rowClass}>{row}</div>}
    </li>
  );
};
