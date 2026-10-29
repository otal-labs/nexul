import { GitBranchIcon } from "lucide-react";
import { Link } from "react-router";

import { MonoRef } from "@/components/MonoRef";
import { DeployStatusBadge } from "@/components/service/DeployStatusBadge";
import { formatRelativeTime } from "@/components/service/DeployTime";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useNow } from "@/hooks/useNow";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { cn } from "@/lib/utils";
import { deployCommit, deployPath, type Deploy } from "@/models/Stack";
import { elapsedShort, imageShortName } from "@/utils/ImageRefUtility";
import { formatColumnTime } from "@/utils/TimeUtility";

interface DeployRowProps {
  deploy: Deploy;
  // The deploy whose image runs now, marked so the list answers "what is live" without a second look.
  current: boolean;
}

// "current" leads the source, so it never takes width from the tag it marks.
const sourceOf = (deploy: Deploy, current: boolean): string =>
  [current && "current", deploy.triggered_by, deploy.pr_number && `#${deploy.pr_number}`, deploy.rule_name && `rule ${deploy.rule_name}`, deploy.kind === "build" && !deploy.triggered_by && !deploy.rule_name && "build"]
    .filter(Boolean)
    .join(" · ");

// What the deploy put on the machine: the image by its last segment, else the ref a build started from.
const What = ({ deploy }: { deploy: Deploy }) => {
  const commit = deployCommit(deploy);
  return (
    <span className="flex min-w-0 items-center gap-2 font-mono text-[13px]">
      {deploy.image && <MonoRef value={imageShortName(deploy.image)} kind="image" title={deploy.image} className="flex-1" />}
      {!deploy.image && (
        <span className="flex min-w-0 items-center gap-1" title={[deploy.ref, commit].filter(Boolean).join(" ")}>
          <GitBranchIcon className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
          <span className="truncate">{deploy.ref || "repository"}</span>
          {commit && <span className="shrink-0 text-muted-foreground">{commit.slice(0, 7)}</span>}
        </span>
      )}
    </span>
  );
};

// Status leads in a fixed column so the image after it never shifts between succeeded and failed.
export const DeployRow = ({ deploy, current }: DeployRowProps) => {
  const canOpen = useAreaAccess()?.("deploys") ?? false;
  const wsPath = useWorkspacePath();
  const inFlight = deploy.status === "running" || deploy.status === "pending";
  const now = useNow(inFlight);
  const source = sourceOf(deploy, current);
  const rowClass =
    "grid min-h-11 grid-cols-[8.5rem_minmax(0,1fr)_7rem] items-center gap-x-4 py-2.5 text-sm";
  const time = (
    <time dateTime={deploy.created_at} title={formatRelativeTime(deploy.created_at)} className="text-right font-mono text-xs whitespace-nowrap text-muted-foreground tabular-nums">
      {formatColumnTime(deploy.created_at)}
    </time>
  );
  const row = (
    <>
      <span className="flex items-center gap-1.5 whitespace-nowrap">
        <DeployStatusBadge status={deploy.status} />
        {inFlight && <span className="font-mono text-xs text-muted-foreground tabular-nums">{`· ${elapsedShort(deploy.created_at, now)}`}</span>}
      </span>
      {/* The source sits under what was deployed, so a short tag leaves no empty column and a long rule keeps the width. */}
      <span className="min-w-0 space-y-0.5">
        <What deploy={deploy} />
        {source && <span className="block truncate font-mono text-xs text-muted-foreground" title={source}>{source}</span>}
      </span>
      {time}
    </>
  );

  return (
    <li>
      {canOpen && (
        <Link to={wsPath(deployPath(deploy))} className={cn(rowClass, "inset-focus rounded-sm px-2 transition-colors duration-150 ease-standard hover:bg-accent/40")}>
          {row}
        </Link>
      )}
      {!canOpen && <div className={cn(rowClass, "px-2")}>{row}</div>}
    </li>
  );
};
