import { Pencil, Send, Trash2 } from "lucide-react";

import { REPEAT_LABELS } from "@nexul/client-core/calendar";
import type { ScheduledMessage } from "@nexul/client-core/chat";

import { Button } from "@/components/ui/button";
import { useCancelScheduledMessage, useSendScheduledMessageNow } from "@/hooks/ScheduledMessageHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useScheduleFormDialog } from "@/hooks/useScheduleFormDialog";
import { formatAheadTime } from "@/utils/TimeUtility";

// One scheduled message: when it posts and how often, its text, and the ways to change, post or cancel it.
export const ScheduledMessageRow = ({ scheduled }: { scheduled: ScheduledMessage }) => {
  const { edit } = useScheduleFormDialog(scheduled.conversation_id);
  const sendNow = useSendScheduledMessageNow();
  const cancel = useCancelScheduledMessage();
  const { open: confirm } = useConfirmationDialog();
  const repeat = scheduled.repeats === "none" ? "" : ` · ${REPEAT_LABELS[scheduled.repeats]}`;

  const remove = async () => {
    if (await confirm({ message: "It won't be posted.", title: "Cancel this scheduled message?", confirmLabel: "Cancel message", cancelLabel: "Keep it" })) {
      cancel.mutate(scheduled);
    }
  };

  return (
    <li className="group flex items-start gap-2 px-3 py-2">
      <div className="min-w-0 flex-1 space-y-0.5">
        <p className="text-xs text-muted-foreground">{formatAheadTime(scheduled.send_at)}{repeat}</p>
        <p className="line-clamp-3 text-sm break-words whitespace-pre-wrap">{scheduled.body}</p>
      </div>
      <div className="flex shrink-0">
        <Button size="icon" variant="ghost" className="size-7" aria-label="Edit scheduled message" onClick={() => void edit(scheduled)}>
          <Pencil className="size-3.5" aria-hidden />
        </Button>
        <Button size="icon" variant="ghost" className="size-7" aria-label="Send now" loading={sendNow.isPending} onClick={() => sendNow.mutate(scheduled)}>
          <Send className="size-3.5" aria-hidden />
        </Button>
        <Button size="icon" variant="ghost" className="size-7" aria-label="Cancel scheduled message" onClick={() => void remove()}>
          <Trash2 className="size-3.5" aria-hidden />
        </Button>
      </div>
    </li>
  );
};
