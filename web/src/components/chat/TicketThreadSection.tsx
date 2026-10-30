import { MessageSquare } from "lucide-react";
import { useState } from "react";

import { ConversationThread } from "@/components/chat/ConversationThread";
import { QueueEventList } from "@/components/chat/QueueEventList";
import { TicketThreadMenu } from "@/components/chat/TicketThreadMenu";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Button } from "@/components/ui/button";
import { useFetchOrCreateTicketThread, useFetchTicketThreadStatus } from "@/hooks/ChatHooks";
import { cn } from "@/lib/utils";

interface TicketThreadSectionProps {
  workspaceId: string;
  ticketId: string;
  /** The thread fills the ticket page's Thread tab, composer at its foot; the tab names it, so it drops its heading. */
  pane?: boolean;
}

// "starting" is local UI state for the one-shot "Start thread" click; an existing thread loads on its own.
export const TicketThreadSection = ({ workspaceId, ticketId, pane = false }: TicketThreadSectionProps) => {
  const [starting, setStarting] = useState(false);
  const { data: status, isPending: statusPending } = useFetchTicketThreadStatus(ticketId);
  const hasThread = status?.[ticketId] === true;
  const shouldLoad = hasThread || starting;
  const showStart = !statusPending && !shouldLoad;
  const { data: conversation, error, isPending } = useFetchOrCreateTicketThread(workspaceId, ticketId, shouldLoad);

  return (
    <div
      className={cn(
        "space-y-3 border-t border-border pt-6",
        pane && "flex min-h-0 flex-1 flex-col border-t-0 pt-0",
      )}
    >
      {!pane && (
        <div className="flex shrink-0 items-center justify-between gap-2">
          <h2 className="text-sm font-semibold tracking-tight">Thread</h2>
          <span className="-my-1.5 ml-auto">
            <TicketThreadMenu workspaceId={workspaceId} ticketId={ticketId} />
          </span>
        </div>
      )}
      {statusPending && <LoadingDisplay label="Loading thread…" />}
      {showStart && <QueueEventList targetType="ticket" targetId={ticketId} />}
      {showStart && pane && <p className="text-sm text-muted-foreground">No messages yet.</p>}
      {showStart && (
        <Button variant="outline" size="sm" className={cn(pane && "w-full")} onClick={() => setStarting(true)}>
          <MessageSquare className="size-4" aria-hidden />
          Start thread
        </Button>
      )}
      {shouldLoad && isPending && <LoadingDisplay label="Loading thread…" />}
      {shouldLoad && error && <ErrorDisplay error={error} title="Couldn't load the thread." />}
      {conversation && (
        <div
          className={cn(
            "h-96 overflow-hidden rounded-lg border border-border",
            pane && "-mx-3 h-auto min-h-0 flex-1 rounded-none border-0 [--chat-column:50rem]",
          )}
        >
          <ConversationThread workspaceId={workspaceId} conversation={conversation} showHeader={false} />
        </div>
      )}
    </div>
  );
};
