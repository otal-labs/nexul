import { useElapsedSeconds } from "@/hooks/useElapsedSeconds";
import { useVoiceCallStore } from "@/stores/voiceCallStore";
import { formatCallElapsed } from "@/utils/TimeUtility";

// Its own leaf so the once-a-second tick re-renders the clock, not the controls.
export const VoiceCallTimer = () => {
  const connectedAt = useVoiceCallStore((s) => s.connectedAt);
  const seconds = useElapsedSeconds(connectedAt ?? 0, connectedAt !== null);
  if (connectedAt === null) return null;
  return (
    <span role="timer" aria-label="Time in call" className="justify-self-start font-mono text-xs tabular-nums text-muted-foreground">
      {formatCallElapsed(seconds)}
    </span>
  );
};
