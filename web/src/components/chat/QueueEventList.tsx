import { QueueEventRow } from "@/components/chat/QueueEventLine";
import { useQueueEvents } from "@/hooks/PlayQueueHooks";
import type { PlayType } from "@/models/Play";
import { placeQueueEvents } from "@/utils/PlayQueueUtility";

interface QueueEventListProps {
  targetType: PlayType;
  targetId: string;
}

// A target's skipped and didn't-run lines while it has no thread to place them in.
export const QueueEventList = ({ targetType, targetId }: QueueEventListProps) => {
  const events = useQueueEvents(targetType, targetId);
  const { opensDay } = placeQueueEvents([], events);

  return (
    events.length > 0 && (
      <div>
        {events.map((event) => (
          <QueueEventRow key={event.item.id} event={event} newDay={opensDay.has(event.item.id)} />
        ))}
      </div>
    )
  );
};
