import { ChevronRight, GitBranchIcon } from "lucide-react";
import { Link } from "react-router";

import { MonoRef } from "@/components/MonoRef";
import { DeployStatusBadge } from "@/components/service/DeployStatusBadge";
import { formatRelativeTime } from "@/components/service/DeployTime";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useNow } from "@/hooks/useNow";
import { useActorNamer } from "@/hooks/usePersonNamer";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { cn } from "@/lib/utils";
import { deployCommit, deployPath, type Deploy } from "@/models/Stack";
import { formatStepDuration } from "@/utils/DeployLogUtility";
import { elapsedShort, imageShortName } from "@/utils/ImageRefUtility";
import { formatColumnTime } from "@/utils/TimeUtility";

interface DeployRowProps {
  deploy: Deploy;
  // The deploy whose image runs now, marked so the list answers "what is live" without a second look.
  current: boolean;
}

const sourceOf = (deploy: Deploy, actor: string | undefined): string =>
  [actor, deploy.pr_number && `#${deploy.pr_number}`, deploy.rule_name && `rule ${deploy.rule_name}`, deploy.kind === "build" && !deploy.triggered_by && !deploy.rule_name && "build"]
    .filter(Boolean)
    .join(" · ");

// What the deploy put on the machine: the image by its last segment, else the ref a build started from; the one running
// now carries a "live" tag, the word the stack's status strip uses for it.
const What = ({ deploy, current }: { deploy: Deploy; current: boolean }) => {
  const commit = deployCommit(deploy);
  return (
    <span className="flex min-w-0 items-center gap-2 font-mono text-row">
      {deploy.image && <MonoRef value={imageShortName(deploy.image)} kind="image" title={deploy.image} className="min-w-0 shrink" />}
      {!deploy.image && (
        <span className="flex min-w-0 items-center gap-1" title={`${deploy.ref} ${commit}`.trim()}>
          <GitBranchIcon className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
          <span className="truncate">{deploy.ref || "repository"}</span>
          {commit && <span className="shrink-0 text-muted-foreground">{commit.slice(0, 7)}</span>}
        </span>
      )}
      {current && (
        <span className="inline-flex shrink-0 items-center gap-1.5 rounded-full border border-border bg-foreground/[0.04] px-2 font-sans text-micro font-medium text-muted-foreground">
          <span aria-hidden className="size-1.5 rounded-full bg-success" />
          live
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
  const actorName = useActorNamer();
  const source = sourceOf(deploy, actorName(deploy.triggered_by));
  const took = inFlight ? "" : formatStepDuration(Date.parse(deploy.updated_at) - Date.parse(deploy.created_at));
  // Below lg the actor column steps aside so the version and commit keep their room; the deploy page names the actor.
  const rowClass = "grid min-h-11 grid-cols-[6.5rem_minmax(0,1fr)_4rem_7rem_1rem] items-center gap-x-4 py-2.5 text-sm lg:grid-cols-[6.5rem_minmax(0,1.2fr)_minmax(0,1fr)_4rem_7rem_1rem]";
  const time = (
    <time dateTime={deploy.created_at} title={formatRelativeTime(deploy.created_at)} className="text-right font-mono text-xs whitespace-nowrap text-muted-foreground tabular-nums">
      {formatColumnTime(deploy.created_at)}
    </time>
  );
  const row = (
    <>
      <span className="flex items-center gap-1.5 whitespace-nowrap">
        {/* A success is the expected outcome and steps back; anything else keeps ink. */}
        <DeployStatusBadge status={deploy.status} className={cn("text-row", deploy.status === "healthy" ? "text-muted-foreground" : "text-foreground")} />
        {inFlight && <span className="font-mono text-xs text-muted-foreground tabular-nums">{`· ${elapsedShort(deploy.created_at, now)}`}</span>}
      </span>
      <What deploy={deploy} current={current} />
      {/* Who and why on every row, in its own column, so each row reads across at one height. */}
      <span className="hidden truncate font-mono text-xs text-muted-foreground lg:block" title={source}>
        {source}
      </span>
      <span title={took && `Took ${took}`} className="text-right font-mono text-xs whitespace-nowrap text-muted-foreground tabular-nums">
        {took}
      </span>
      {time}
      {canOpen && <ChevronRight aria-hidden className="size-4 text-muted-foreground" />}
    </>
  );

  return (
    <li>
      {canOpen && (
        <Link to={wsPath(deployPath(deploy))} className={cn(rowClass, "inset-focus rounded-sm px-2 transition-colors duration-150 ease-standard hover:bg-accent")}>
          {row}
        </Link>
      )}
      {!canOpen && <div className={cn(rowClass, "px-2")}>{row}</div>}
    </li>
  );
};
