import { microheaderClass } from "@/components/Microheader";
import { AutoPlaySignals } from "@/components/play/AutoPlaySignals";
import { useFetchPlayQueue } from "@/hooks/PlayQueueHooks";
import { cn } from "@/lib/utils";
import { hasAutoPlaySignals } from "@/utils/PlayQueueUtility";

interface DocAutoPlaysSectionProps {
  docId: string;
  className?: string;
}

// A doc's queued and paused auto plays, beside its trail; only there while something waits or the cap paused it.
export const DocAutoPlaysSection = ({ docId, className }: DocAutoPlaysSectionProps) => {
  const { data: queue } = useFetchPlayQueue("doc", docId);

  return (
    hasAutoPlaySignals(queue) && (
      <section className={cn("space-y-0.5", className)}>
        <h2 className={cn(microheaderClass, "px-2 pb-1")}>Auto plays</h2>
        <AutoPlaySignals targetType="doc" targetId={docId} developer="" />
      </section>
    )
  );
};
