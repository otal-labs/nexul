import { Clock } from "lucide-react";
import { useState } from "react";

import { personLabel } from "@nexul/client-core/person";

import { Button } from "@/components/ui/button";
import { signalActionClass, signalRowClass } from "@/components/play/signalRowClass";
import { useFetchMe } from "@/hooks/AuthHooks";
import { usePerson } from "@/hooks/PeopleHooks";
import { useCancelQueuedRun } from "@/hooks/PlayQueueHooks";
import { leavingRowClass } from "@/hooks/useRowGlide";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import type { PlayQueueItem } from "@/models/PlayQueue";
import { cn } from "@/lib/utils";
import { waitingWhy } from "@/utils/PlayQueueUtility";

interface QueuedRunRowProps {
  item: PlayQueueItem;
  // Called as the row starts to leave, so the list can glide the rows under it up once it is gone.
  onLeave: () => void;
}

// Cancel is for the person the run lands on or an autoplays:write holder, as the server checks.
export const QueuedRunRow = ({ item, onLeave }: QueuedRunRowProps) => {
  const person = personLabel(usePerson(item.person_id));
  const { data: me } = useFetchMe();
  const canWrite = useHasPermission("autoplays:write");
  const cancel = useCancelQueuedRun();
  const [leaving, setLeaving] = useState(false);
  const canCancel = item.status === "queued" && (me?.user.id === item.person_id || canWrite);

  const onCancel = () => {
    setLeaving(true);
    onLeave();
    cancel.mutate(item.id, { onError: () => setLeaving(false) });
  };

  return (
    <li data-leaving={leaving || undefined} className={cn(leavingRowClass, signalRowClass)}>
      <Clock className="mt-0.5 size-3.5 shrink-0 text-info" aria-hidden />
      <div className="min-w-0 flex-1">
        <p className="[overflow-wrap:anywhere]">{item.play_label} queued</p>
        <p className="text-muted-foreground [overflow-wrap:anywhere]">
          {person} · {waitingWhy(item)}
        </p>
      </div>
      {canCancel && (
        <Button variant="ghost" size="sm" className={signalActionClass} aria-label={`Cancel ${item.play_label}`} disabled={leaving} onClick={onCancel}>
          Cancel
        </Button>
      )}
    </li>
  );
};
