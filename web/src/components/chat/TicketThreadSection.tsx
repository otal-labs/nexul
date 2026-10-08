import { useQueryClient } from "@tanstack/react-query";
import { MessageSquare } from "lucide-react";
import { useEffect, useState } from "react";

import { ConversationThread } from "@/components/chat/ConversationThread";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { RowActions } from "@/components/listpane/RowActions";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Button } from "@/components/ui/button";
import {
  getChatThreadIndicatorsKey,
  getChatTicketThreadStatusKey,
  useFetchOrCreateTicketThread,
  useFetchTicketThreadStatus,
} from "@/hooks/ChatHooks";
import { useBotsDialog } from "@/hooks/useBotsDialog";
import { cn } from "@/lib/utils";

interface TicketThreadSectionProps {
  workspaceId: string;
  ticketId: string;
  /** The thread is its own full-height panel beside the body, composer at its foot. */
  pane?: boolean;
}

// "starting" is local UI state for the one-shot "Start chat" click; an existing thread loads on its own.
export const TicketThreadSection = ({ workspaceId, ticketId, pane = false }: TicketThreadSectionProps) => {
  const [starting, setStarting] = useState(false);
  const client = useQueryClient();
  const { data: status, isPending: statusPending } = useFetchTicketThreadStatus(ticketId);
  const hasThread = status?.[ticketId] === true;
  const shouldLoad = hasThread || starting;
  const showStart = !statusPending && !shouldLoad;
  const { data: conversation, error, isPending } = useFetchOrCreateTicketThread(workspaceId, ticketId, shouldLoad);
  const { onBots, botsDialog } = useBotsDialog(conversation, "Ticket thread");

  useEffect(() => {
    if (!conversation) return;
    void client.invalidateQueries({ queryKey: [getChatThreadIndicatorsKey] });
    void client.invalidateQueries({ queryKey: [getChatTicketThreadStatusKey, ticketId] });
  }, [conversation, client, ticketId]);

  return (
    <div
      className={cn(
        "space-y-3 border-t border-border pt-6",
        pane &&
          "@min-[46rem]:flex @min-[46rem]:min-h-0 @min-[46rem]:flex-1 @min-[46rem]:flex-col @min-[46rem]:border-t-0 @min-[46rem]:pt-0",
      )}
    >
      <div className="flex shrink-0 items-center justify-between gap-2">
        <h2 className="text-sm font-semibold tracking-tight">Thread</h2>
        {onBots && (
          <span className="-my-1.5">
            <RowActions itemLabel="the thread" onBots={onBots} />
          </span>
        )}
      </div>
      {statusPending && <LoadingDisplay label="Loading thread…" />}
      {showStart && pane && <p className="text-sm text-muted-foreground">No messages yet.</p>}
      {showStart && (
        <Button variant="outline" size="sm" className={cn(pane && "w-full")} onClick={() => setStarting(true)}>
          <MessageSquare className="size-4" aria-hidden />
          Start chat
        </Button>
      )}
      {shouldLoad && isPending && <LoadingDisplay label="Loading thread…" />}
      {shouldLoad && error && <ErrorDisplay error={error} title="Failed to load the thread." />}
      {conversation && (
        <div
          className={cn(
            "h-96 overflow-hidden rounded-lg border border-border",
            pane && "@min-[46rem]:h-auto @min-[46rem]:min-h-0 @min-[46rem]:flex-1 @min-[46rem]:rounded-none @min-[46rem]:border-0",
          )}
        >
          <ConversationThread workspaceId={workspaceId} conversation={conversation} showHeader={false} />
        </div>
      )}
      {botsDialog}
    </div>
  );
};
