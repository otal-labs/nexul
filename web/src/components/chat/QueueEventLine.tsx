import { CircleSlash, TriangleAlert } from "lucide-react";

import { MessageScrollerItem } from "@/components/ui/message-scroller";
import { RunItButton } from "@/components/play/RunItButton";
import { cn } from "@/lib/utils";
import type { ThreadQueueEvent } from "@/utils/PlayQueueUtility";
import { outcomeText } from "@/utils/PlayQueueUtility";
import { formatClockTime, formatFullTime } from "@/utils/TimeUtility";

interface QueueEventLineProps {
  event: ThreadQueueEvent;
  // Decided after the thread opened: it rises in like any message that arrives.
  arrived: boolean;
}

export const QueueEventLine = ({ event: { item, retry }, arrived }: QueueEventLineProps) => {
  const didntRun = item.status === "didnt_run";

  return (
    <MessageScrollerItem messageId={`queue-${item.id}`} className={cn("pt-1.5 [content-visibility:visible]", arrived && "arrive")}>
      <div className="flex items-center gap-2 px-3 py-1">
        <span className="flex w-8 shrink-0 justify-center">
          <span className="flex size-6 items-center justify-center rounded-full border border-border bg-muted/60">
            {didntRun && <TriangleAlert className="size-3 text-warning" aria-hidden />}
            {!didntRun && <CircleSlash className="size-3 text-muted-foreground" aria-hidden />}
          </span>
        </span>
        <p className="min-w-0 flex-1 text-xs text-muted-foreground [overflow-wrap:anywhere]">
          <span className="font-medium text-foreground">{item.play_label}</span> {didntRun ? "didn't run" : "skipped"}: {outcomeText(item.reason)}{" "}
          <time dateTime={item.decided_at} title={formatFullTime(item.decided_at)} className="font-mono whitespace-nowrap tabular-nums">
            {formatClockTime(item.decided_at)}
          </time>
        </p>
        {retry && <RunItButton item={item} />}
      </div>
    </MessageScrollerItem>
  );
};
