import { CircleCheckIcon, CircleHelpIcon, CircleMinusIcon, CircleXIcon, LoaderIcon, type LucideIcon } from "lucide-react";
import { Link } from "react-router";

import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { PLAY_STAGE_LABELS } from "@/models/Play";
import type { SweepLine } from "@/models/Schedule";
import { cn } from "@/lib/utils";

interface LineLook {
  icon: LucideIcon;
  tone: string;
  word: string;
}

// How a ticket went, as an icon in its status hue beside plain words.
const lookOf = (line: SweepLine): LineLook => {
  if (line.outcome === "finished") return { icon: CircleCheckIcon, tone: "text-success", word: "Finished" };
  if (line.outcome === "asked") return { icon: CircleHelpIcon, tone: "text-warning", word: "Asked" };
  if (line.outcome === "failed") return { icon: CircleXIcon, tone: "text-destructive", word: "Failed" };
  if (line.outcome === "running" || line.status === "queued" || line.status === "dispatching")
    return { icon: LoaderIcon, tone: "text-muted-foreground", word: line.outcome === "running" ? "Running" : "Waiting" };
  if (line.status === "didnt_run") return { icon: CircleXIcon, tone: "text-muted-foreground", word: "Didn't run" };
  return { icon: CircleMinusIcon, tone: "text-muted-foreground", word: "Skipped" };
};

// The why under the outcome: the question a run asked, or why a ticket was skipped or didn't run.
const detailOf = (line: SweepLine) => {
  if (line.outcome === "asked") return line.question;
  if (line.outcome === "") return line.reason;
  return "";
};

export const SweepLineRow = ({ line }: { line: SweepLine }) => {
  const wsPath = useWorkspacePath();
  const look = lookOf(line);
  const detail = detailOf(line);
  return (
    <li className="flex items-start gap-3 border-t border-border py-2.5 first:border-t-0">
      <look.icon className={cn("mt-0.5 size-4 shrink-0", look.tone)} aria-hidden />
      <div className="min-w-0 flex-1 space-y-0.5">
        <Link to={wsPath(`/tickets/${line.key}`)} className="flex min-w-0 items-baseline gap-2 text-row hover:underline">
          <span className="technical shrink-0 text-xs text-muted-foreground">{line.key}</span>
          <span className="truncate text-foreground" title={line.title}>
            {line.title || line.key}
          </span>
        </Link>
        <p className="text-xs text-pretty text-muted-foreground [overflow-wrap:anywhere]">
          <span className="text-foreground">{look.word}</span>
          {detail && `: ${detail}`}
          {line.outcome !== "" && line.stage && `${/[.?!]$/.test(detail) ? "" : "."} Now in ${PLAY_STAGE_LABELS[line.stage]}`}
        </p>
        {line.bugs.length > 0 && (
          <p className="text-xs text-muted-foreground [overflow-wrap:anywhere]">
            Found{" "}
            {line.bugs.map((bug, i) => (
              <span key={bug.key}>
                {i > 0 && ", "}
                <span className="technical">{bug.key}</span>: {bug.title}
              </span>
            ))}
          </p>
        )}
      </div>
    </li>
  );
};
