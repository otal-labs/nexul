import { cardElapsed } from "@/components/board/cardElapsed";
import { useElapsedSeconds } from "@/hooks/useElapsedSeconds";

interface RunTimerProps {
  startedAt: string;
}

// Its own leaf so the once-a-second tick re-renders the timer, not the memoized card body.
export const RunTimer = ({ startedAt }: RunTimerProps) => {
  const seconds = useElapsedSeconds(Date.parse(startedAt));
  return <span className="tabular-nums">{cardElapsed(seconds)}</span>;
};
