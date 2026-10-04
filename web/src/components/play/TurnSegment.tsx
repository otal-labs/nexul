import { TrailReplyProse } from "@/components/play/TrailReplyProse";
import { TrailTurnGroup } from "@/components/play/TrailTurnGroup";
import type { TranscriptSegment } from "@/utils/TrailTranscriptUtility";

interface TurnSegmentProps {
  segment: TranscriptSegment;
}

// The group is keyed by whether it runs, so it folds shut once the agent moves on or the turn ends.
export const TurnSegment = ({ segment }: TurnSegmentProps) => (
  <>
    {segment.kind === "turn" && (
      <TrailTurnGroup key={String(segment.running)} entries={segment.entries} running={segment.running} from={segment.from} until={segment.until} />
    )}
    {segment.kind === "reply" && <TrailReplyProse text={segment.entry.detail || segment.entry.summary} />}
  </>
);
