import { useElapsedSeconds } from "@/hooks/useElapsedSeconds";
import { formatElapsed } from "@/utils/TrailTranscriptUtility";

interface RunTimerProps {
  startedAt: string;
}

// Its own leaf so the once-a-second tick re-renders the timer, not the memoized card body.
export const RunTimer = ({ startedAt }: RunTimerProps) => {
  const seconds = useElapsedSeconds(Date.parse(startedAt));
  return <span className="tabular-nums">{formatElapsed(seconds)}</span>;
};
