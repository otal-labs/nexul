import { TrailTurnGroup } from "@/components/play/TrailTurnGroup";
import type { TrailTurn } from "@/utils/ThreadTrailUtility";

interface MessageTrailTurnsProps {
  turns: TrailTurn[];
}

// The Agent's work above a thread message, in the trail dialog's own turn groups.
export const MessageTrailTurns = ({ turns }: MessageTrailTurnsProps) => (
  <div className="flex flex-col">
    {turns.map((turn, i) => (
      <TrailTurnGroup key={`${turn.from ?? ""}-${i}`} entries={turn.entries} running={turn.running} from={turn.from} until={turn.until} />
    ))}
  </div>
);
