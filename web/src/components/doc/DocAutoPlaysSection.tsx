import { microheaderClass } from "@/components/Microheader";
import { QueueEventList } from "@/components/chat/QueueEventList";
import { AutoPlaySignals } from "@/components/play/AutoPlaySignals";
import { useFetchConversations } from "@/hooks/ChatHooks";
import { useFetchPlayQueue, useQueueEvents } from "@/hooks/PlayQueueHooks";
import { cn } from "@/lib/utils";
import { hasAutoPlaySignals } from "@/utils/PlayQueueUtility";

interface DocAutoPlaysSectionProps {
  workspaceId: string;
  docId: string;
  className?: string;
}

// A doc's queued and paused auto plays beside its trail, and, until the doc has a thread to hold them, what was skipped or didn't run.
export const DocAutoPlaysSection = ({ workspaceId, docId, className }: DocAutoPlaysSectionProps) => {
  const { data: queue } = useFetchPlayQueue("doc", docId);
  const events = useQueueEvents("doc", docId);
  const { data: conversations } = useFetchConversations(workspaceId);
  const threadless = !!conversations && !conversations.some((c) => c.kind === "doc_thread" && c.doc_id === docId);
  const showEvents = threadless && events.length > 0;

  return (
    (hasAutoPlaySignals(queue) || showEvents) && (
      <section className={cn("space-y-0.5", className)}>
        <h2 className={cn(microheaderClass, "px-2 pb-1")}>Auto plays</h2>
        <AutoPlaySignals targetType="doc" targetId={docId} developer="" />
        {showEvents && <QueueEventList targetType="doc" targetId={docId} />}
      </section>
    )
  );
};
