import { TrailStateIcon } from "@/components/play/TrailStateIcon";
import { usePlayRunStore } from "@/stores/playRunStore";
import { answersChangedSinceWritten, clarificationStatus, type Clarification } from "@/models/DocClarification";
import { isTrailActive } from "@/models/Trail";
import { cn } from "@/lib/utils";

interface ClarificationStatusLineProps {
  clarification: Clarification;
  locked: boolean;
}

// The clarification's state as one line under the heading; people who may close it also see answers that changed
// since a round wrote the doc.
export const ClarificationStatusLine = ({ clarification, locked }: ClarificationStatusLineProps) => {
  const dev = clarification.can_close;
  const status = clarificationStatus(clarification, dev, locked);
  const trailId = clarification.rounds.at(-1)?.trail_id ?? "";
  const live = usePlayRunStore((s) => s.frames[trailId]?.state);
  const icon = status.icon === "running" && live && isTrailActive(live) ? live : status.icon;
  const changed = dev && !clarification.running ? answersChangedSinceWritten(clarification) : 0;
  return (
    <>
      <p className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-0.5 text-sm">
        {icon && <TrailStateIcon state={icon} />}
        <span className={cn(icon ? "font-medium" : "text-muted-foreground")}>{status.label}</span>
        {status.detail !== "" && <span className="text-muted-foreground">· {status.detail}</span>}
      </p>
      {changed > 0 && (
        <p className="flex items-center gap-2 text-sm text-muted-foreground">
          <span role="img" aria-label="out of date" className="flex size-3.5 shrink-0 items-center justify-center">
            <span className="size-2 rounded-full bg-warning" />
          </span>
          {changed} {changed === 1 ? "answer" : "answers"} changed since the doc was written
        </p>
      )}
    </>
  );
};
