import { TrailQuestionCard } from "@/components/play/TrailQuestionCard";
import { TrailReplyProse } from "@/components/play/TrailReplyProse";
import { TrailTurnGroup } from "@/components/play/TrailTurnGroup";
import { TrailUserBubble } from "@/components/play/TrailUserBubble";
import type { TranscriptSegment } from "@/utils/TrailTranscriptUtility";

interface TrailTranscriptSegmentProps {
  trailId: string;
  segment: TranscriptSegment;
}

// One block of the conversation; the question card reads its own trail from the cache, so only the id travels.
export const TrailTranscriptSegment = ({ trailId, segment }: TrailTranscriptSegmentProps) => (
  <>
    {segment.kind === "user" && <TrailUserBubble body={segment.body} at={segment.at} />}
    {segment.kind === "turn" && <TrailTurnGroup entries={segment.entries} running={segment.running} from={segment.from} until={segment.until} />}
    {segment.kind === "question" && (
      <div className="px-1 py-1">
        <TrailQuestionCard trailId={trailId} />
      </div>
    )}
    {segment.kind === "reply" && <TrailReplyProse text={segment.entry.detail || segment.entry.summary} />}
    {segment.kind === "note" && <p className="px-3 py-1 text-xs text-muted-foreground italic">{segment.text}</p>}
  </>
);
