import { HealthDot } from "@/components/service/HealthDot";
import { formatRelativeTime } from "@/components/service/DeployTime";
import type { Deploy } from "@/models/Stack";

interface DeployRowProps {
  deploy: Deploy;
}

// Shows the image, not a build/deploy log — no deploy-log route exists yet, so this is the closest truthful reference.
export const DeployRow = ({ deploy }: DeployRowProps) => (
  <li className="flex items-center gap-3 px-3 py-2.5 text-sm">
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
  </li>
);
