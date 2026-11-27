import { QueuedRunRow } from "@/components/plays/QueuedRunRow";
import { SettingsCard } from "@/components/SettingsCard";
import { useFetchPlayQueued } from "@/hooks/PlayQueueHooks";

// What this play has waiting in the queue, each with Cancel.
export const PlayQueuedSection = ({ playId }: { playId: string }) => {
  const { data: queued } = useFetchPlayQueued(playId);
  return (
    queued &&
    queued.length > 0 && (
      <SettingsCard title="Queued">
        {queued.map((item, i) => (
          <QueuedRunRow key={item.id} item={item} title={`Run on a ${item.target_type}`} first={i === 0} />
        ))}
      </SettingsCard>
    )
  );
};
