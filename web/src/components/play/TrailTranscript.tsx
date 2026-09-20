import { TrailTranscriptSegment } from "@/components/play/TrailTranscriptSegment";
import type { ActivityEntry, Trail, TrailQuestion, TrailState } from "@/models/Trail";
import { segmentTranscript } from "@/utils/TrailTranscriptUtility";

interface TrailTranscriptProps {
  trail: Trail;
  steps: ActivityEntry[];
  state: TrailState;
  question: TrailQuestion | null;
}

// The run as a conversation: the starter's bubbles on the right, the Agent's turn groups and final reply on the
// left, the question card and the runner's notes where they happened. The container scrolls, the blocks never do.
export const TrailTranscript = ({ trail, steps, state, question }: TrailTranscriptProps) => {
  const segments = segmentTranscript(trail, steps, state, question);
  return (
    <div className="flex flex-col gap-1">
      {segments.map((segment, i) => (
        <TrailTranscriptSegment key={`${segment.kind}-${i}`} trailId={trail.id} segment={segment} />
      ))}
    </div>
  );
};
