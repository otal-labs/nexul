import type { StepSegment } from "@nexul/client-core/agentSteps";

import { TrailReplyProse } from "@/components/play/TrailReplyProse";
import { TurnRunLine } from "@/components/play/TurnRunLine";
import type { ActivityEntry } from "@/models/Trail";

interface TurnFeedItemProps {
  segment: StepSegment<ActivityEntry>;
  root: string | null;
}

// One entry of a turn's feed: a sentence the Agent said, or the tools it ran before the next one.
export const TurnFeedItem = ({ segment, root }: TurnFeedItemProps) => (
  <>
    {segment.kind === "reply" && <TrailReplyProse text={segment.entry.detail || segment.entry.summary} />}
    {segment.kind === "turn" && <TurnRunLine entries={segment.entries} running={segment.running} root={root} />}
  </>
);
