import { AutoPlaysPausedRow } from "@/components/play/AutoPlaysPausedRow";
import { QueuedRunRow } from "@/components/play/QueuedRunRow";
import { EnterList } from "@/components/EnterList";
import { useFetchPlayQueue } from "@/hooks/PlayQueueHooks";
import { useRowGlide } from "@/hooks/useRowGlide";
import type { PlayType } from "@/models/Play";
import { hasAutoPlaySignals, waitingRuns } from "@/utils/PlayQueueUtility";

interface AutoPlaySignalsProps {
  targetType: PlayType;
  targetId: string;
  developer: string;
}

// What auto plays have waiting here now, and whether the daily cap paused them; what already happened is in the thread.
export const AutoPlaySignals = ({ targetType, targetId, developer }: AutoPlaySignalsProps) => {
  const { data: queue } = useFetchPlayQueue(targetType, targetId);
  const { ref: glideRef, prepare } = useRowGlide();

  return (
    queue &&
    hasAutoPlaySignals(queue) && (
      <div ref={glideRef}>
        <EnterList className="flex flex-col">
          {waitingRuns(queue).map((item) => (
            <QueuedRunRow key={item.id} item={item} onLeave={prepare} />
          ))}
          {queue.paused && (
            <AutoPlaysPausedRow
              targetType={targetType}
              targetId={targetId}
              autoRuns={queue.auto_runs}
              developer={developer}
              onLeave={prepare}
            />
          )}
        </EnterList>
      </div>
    )
  );
};
