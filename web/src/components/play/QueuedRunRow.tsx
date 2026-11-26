import { Clock } from "lucide-react";
import { useState } from "react";

import { personLabel } from "@nexul/client-core/person";
import { waitingLines } from "@nexul/client-core/playQueue";

import { Button } from "@/components/ui/button";
import { ReadinessFixLink } from "@/components/pairing/ReadinessFixLink";
import { WaitingCheckoutFix } from "@/components/play/WaitingCheckoutFix";
import { signalActionClass, signalRowClass } from "@/components/play/signalRowClass";
import { useFetchMe } from "@/hooks/AuthHooks";
import { usePerson } from "@/hooks/PeopleHooks";
import { useCancelQueuedRun } from "@/hooks/PlayQueueHooks";
import { leavingRowClass } from "@/hooks/useRowGlide";
import { useActionState } from "@/hooks/useActionState";
import { checkoutPasses } from "@/models/Pairing";
import type { PlayQueueItem } from "@/models/PlayQueue";
import { cn } from "@/lib/utils";
import { waitingLink } from "@/utils/PlayQueueUtility";

interface QueuedRunRowProps {
  item: PlayQueueItem;
  // Called as the row starts to leave, so the list can glide the rows under it up once it is gone.
  onLeave: () => void;
}

// Cancel is for the person the run lands on or an autoplays:write holder, as the server checks.
export const QueuedRunRow = ({ item, onLeave }: QueuedRunRowProps) => {
  const person = personLabel(usePerson(item.person_id));
  const { data: me } = useFetchMe();
  const canWrite = useActionState({ permission: "autoplays:write" }).state === "live";
  const cancel = useCancelQueuedRun();
  const [leaving, setLeaving] = useState(false);
  const mine = me?.user.id === item.person_id;
  const fixes = mine && item.status === "queued" ? checkoutPasses(item.waiting_on ?? []) : [];
  const link = fixes.length === 0 ? waitingLink(item) : undefined;
  const lines = waitingLines(item);
  const canCancel = item.status === "queued" && (mine || canWrite);

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
        {lines.map((line, i) => (
          <p key={i} className="text-muted-foreground [overflow-wrap:anywhere]">
            {i === 0 && `${person} · `}
            {line}
            {i === lines.length - 1 && mine && link && (
              <>
                {" · "}
                <ReadinessFixLink {...link} />
              </>
            )}
          </p>
        ))}
        {fixes.map((pass) => (
          <WaitingCheckoutFix key={pass.computer_id} projectId={item.project_id} pass={pass} />
        ))}
      </div>
      {canCancel && (
        <Button variant="ghost" size="sm" className={signalActionClass} aria-label={`Cancel ${item.play_label}`} disabled={leaving} onClick={onCancel}>
          Cancel
        </Button>
      )}
    </li>
  );
};
