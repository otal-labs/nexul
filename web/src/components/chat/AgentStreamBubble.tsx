import { Bot, Square } from "lucide-react";

import { MessageTrailTurns } from "@/components/chat/MessageTrailTurns";
import { TrailActionRow } from "@/components/play/TrailActionRow";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useElapsedSeconds } from "@/hooks/useElapsedSeconds";
import type { ActivityEntry } from "@/models/Trail";
import type { AgentStreamFrame } from "@/stores/agentStreamStore";
import type { TrailBlock } from "@/utils/ThreadTrailUtility";

interface AgentStreamBubbleProps {
  frame: AgentStreamFrame;
  onInterrupt: () => void;
  // live is the play run behind this turn, whose open turn group shows every step so far; null for a plain @Agent turn.
  live: TrailBlock | null;
}

// The turn group carries its own clock, so the header only speaks up once the reply starts streaming.
const streamingLabel = (frame: AgentStreamFrame, elapsed: number, hasTurns: boolean): string | null => {
  if (frame.text) return "is replying…";
  if (hasTurns) return null;
  return `Working for ${elapsed}s`;
};

// The frame carries the latest step without its detail or time, enough for the same row the trail shows.
const latestStep = (frame: AgentStreamFrame): ActivityEntry => ({
  kind: frame.activityKind ?? "other",
  ...(frame.activityTool && { tool: frame.activityTool }),
  summary: frame.activity,
  at: "",
});

// Mirrors MessageRow's agent-kind shape so it doesn't visually jump when the persisted message replaces it.
export const AgentStreamBubble = ({ frame, onInterrupt, live }: AgentStreamBubbleProps) => {
  const elapsed = useElapsedSeconds(frame.startedAt);
  const label = streamingLabel(frame, elapsed, live !== null);
  return (
    <div className="flex gap-2.5 px-3 py-1.5">
      <div className="mt-0.5 flex size-6 shrink-0 items-center justify-center rounded-full bg-accent text-accent-foreground">
        <Bot className="size-3.5" aria-hidden />
      </div>
      <div className="min-w-0 flex-1 space-y-0.5">
        <div className="flex items-baseline gap-2">
          <span className="truncate text-sm font-semibold">Agent</span>
          <Badge variant="outline" className="h-4 px-1 text-[9px] tracking-wide uppercase">
            App
          </Badge>
          {frame.streaming && label !== null && <span className="truncate text-[11px] text-muted-foreground">{label}</span>}
        </div>
        {live !== null && <MessageTrailTurns turns={live.turns} />}
        {frame.text && <p className="px-1 py-1 text-sm break-words whitespace-pre-wrap">{frame.text}</p>}
        {live === null && !frame.text && frame.activity && (
          <ul className="max-w-[85%]">
            <TrailActionRow entry={latestStep(frame)} live />
          </ul>
        )}
        {live === null && !frame.text && !frame.activity && (
          <p className="max-w-[85%] rounded-2xl rounded-bl-md bg-accent px-3 py-2 text-sm text-muted-foreground animate-pulse">…</p>
        )}
      </div>
      {frame.streaming && (
        <Button size="icon" variant="ghost" className="size-6 shrink-0" aria-label="Stop agent" onClick={onInterrupt}>
          <Square className="size-3.5" aria-hidden />
        </Button>
      )}
    </div>
  );
};
