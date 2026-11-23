import type { ReactNode } from "react";
import { Pencil, Send, Trash2 } from "lucide-react";

import { REPEAT_LABELS } from "@nexul/client-core/calendar";
import type { ScheduledMessage } from "@nexul/client-core/chat";

import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useCancelScheduledMessage, useSendScheduledMessageNow } from "@/hooks/ScheduledMessageHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useScheduleFormDialog } from "@/hooks/useScheduleFormDialog";
import { formatAheadTime } from "@/utils/TimeUtility";

interface RowActionProps {
  label: string;
  loading?: boolean;
  onClick: () => void;
  children: ReactNode;
}

const RowAction = ({ label, loading = false, onClick, children }: RowActionProps) => (
  <Tooltip>
    <TooltipTrigger asChild>
      <Button size="icon" variant="ghost" className="hit-area size-7" aria-label={label} loading={loading} onClick={onClick}>
        {children}
      </Button>
    </TooltipTrigger>
    <TooltipContent>{label}</TooltipContent>
  </Tooltip>
);

// One scheduled message: when it posts and how often, its text, and the ways to change, post or cancel it.
export const ScheduledMessageRow = ({ scheduled }: { scheduled: ScheduledMessage }) => {
  const { edit } = useScheduleFormDialog(scheduled.conversation_id);
  const sendNow = useSendScheduledMessageNow();
  const cancel = useCancelScheduledMessage();
  const { open: confirm } = useConfirmationDialog();
  const repeat = scheduled.repeats === "none" ? "" : ` · ${REPEAT_LABELS[scheduled.repeats]}`;

  const post = async () => {
    const message = scheduled.repeats === "none" ? "It posts here as you." : "It posts here as you. The repeat keeps its next time.";
    if (await confirm({ message, title: "Post it now?", confirmLabel: "Post now", cancelLabel: "Keep it scheduled", destructive: false })) {
      sendNow.mutate(scheduled);
    }
  };

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
      {/* On touch each 28px button takes a 44px hit area, so the gap keeps a missed tap off its neighbour. */}
      <div className="flex shrink-0 pointer-coarse:gap-4">
        <RowAction label="Edit scheduled message" onClick={() => void edit(scheduled)}>
          <Pencil className="size-3.5" aria-hidden />
        </RowAction>
        <RowAction label="Send now" loading={sendNow.isPending} onClick={() => void post()}>
          <Send className="size-3.5" aria-hidden />
        </RowAction>
        <RowAction label="Cancel scheduled message" onClick={() => void remove()}>
          <Trash2 className="size-3.5" aria-hidden />
        </RowAction>
      </div>
    </li>
  );
};
