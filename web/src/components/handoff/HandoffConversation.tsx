import { DialogDescription, DialogTitle } from "@/components/ui/dialog";

import { EmptyRow } from "@/components/EmptyRow";
import { HandoffStateDot } from "@/components/handoff/HandoffStateDot";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { HarnessProviderMark } from "@/components/model/HarnessProviderMark";
import { TrailReplyProse } from "@/components/play/TrailReplyProse";
import { TrailUserBubble } from "@/components/play/TrailUserBubble";
import { TurnSegment } from "@/components/play/TurnSegment";
import { HANDOFF_STATE_LABELS, type Handoff } from "@/models/Handoff";
import { segmentSetupTurn } from "@/utils/TrailTranscriptUtility";

interface HandoffConversationProps {
  handoff: Handoff;
}

// The handed-off agent's conversation: the prompt it was given as the opening bubble, its steps, then its reply.
export const HandoffConversation = ({ handoff }: HandoffConversationProps) => {
  const running = handoff.state === "running";
  const segments = segmentSetupTurn(handoff.steps, running);
  return (
    <div className="flex min-w-0 flex-col gap-4">
      <div className="space-y-2 pr-8">
        <DialogDescription className="flex items-center gap-1.5 font-mono text-xs text-muted-foreground">
          <HarnessProviderMark driver={handoff.driver} className="size-3.5 shrink-0" />
          {handoff.model && <span className="truncate">{handoff.model}</span>}
          <HandoffStateDot state={handoff.state} />
          <span className="shrink-0">{HANDOFF_STATE_LABELS[handoff.state]}</span>
        </DialogDescription>
        <DialogTitle className="text-lg leading-snug font-semibold tracking-tight">{handoff.title}</DialogTitle>
      </div>
      <div className="flex flex-col gap-1">
        <TrailUserBubble body={handoff.prompt} at={null} />
        {segments.map((segment, i) => (
          <TurnSegment key={i} segment={segment} />
        ))}
        {running && handoff.steps.length === 0 && <LoadingDisplay label="Working…" className="justify-start p-1" />}
        {handoff.reply !== "" && <TrailReplyProse text={handoff.reply} />}
        {!running && handoff.reply === "" && (
          <EmptyRow>{handoff.state === "left_running" ? "Still running in T3 Code." : "No reply came back."}</EmptyRow>
        )}
      </div>
    </div>
  );
};
