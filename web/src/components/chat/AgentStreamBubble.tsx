import { Bot, Square } from "lucide-react";

import { MessageTrailTurns } from "@/components/chat/MessageTrailTurns";
import { HandoffPills } from "@/components/handoff/HandoffPills";
import { TrailTurnGroup } from "@/components/play/TrailTurnGroup";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { microheaderClass } from "@/components/Microheader";
import { useElapsedSeconds } from "@/hooks/useElapsedSeconds";
import { cn } from "@/lib/utils";
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
  if (frame.text && !frame.activity) return "is replying…";
  if (hasTurns) return null;
  return `Working for ${elapsed}s`;
};

// Mirrors MessageRow's agent-kind shape so it doesn't visually jump when the persisted message replaces it.
export const AgentStreamBubble = ({ frame, onInterrupt, live }: AgentStreamBubbleProps) => {
  const elapsed = useElapsedSeconds(frame.startedAt);
  // A turn only handing off has nothing of its own to show until its first step or text.
  const showSteps = live === null && (frame.steps.length > 0 || frame.handoffs.length === 0);
  const label = streamingLabel(frame, elapsed, live !== null || showSteps);
  return (
    <div className="flex gap-2.5 px-3 py-1.5">
      <div className="flex size-8 shrink-0 items-center justify-center rounded-full bg-accent text-accent-foreground">
        <Bot className="size-3.5" aria-hidden />
      </div>
      <div className="min-w-0 flex-1 space-y-0.5">
        <div className="flex items-baseline gap-2">
          <span className="truncate text-sm font-semibold">Agent</span>
          <Badge variant="outline" className={cn(microheaderClass, "h-4 shrink-0 px-1")}>
            App
          </Badge>
          {frame.streaming && label !== null && <span className="min-w-0 truncate text-xs text-muted-foreground">{label}</span>}
        </div>
        {live !== null && <MessageTrailTurns turns={live.turns} />}
        {live !== null && frame.text && <p className="px-1 py-1 text-sm break-words whitespace-pre-wrap">{frame.text}</p>}
        {showSteps && <TrailTurnGroup entries={frame.steps} running={frame.streaming} from={new Date(frame.startedAt).toISOString()} until={null} />}
        {frame.handoffs.length > 0 && <HandoffPills handoffs={frame.handoffs} />}
      </div>
      {frame.streaming && (
        <Button size="icon" variant="ghost" className="size-6 shrink-0" aria-label="Stop agent" onClick={onInterrupt}>
          <Square className="size-3.5" aria-hidden />
        </Button>
      )}
    </div>
  );
};
