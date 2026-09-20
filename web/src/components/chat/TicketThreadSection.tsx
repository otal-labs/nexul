import { useQueryClient } from "@tanstack/react-query";
import { MessageSquare } from "lucide-react";
import { useEffect, useState } from "react";

import { ConversationThread } from "@/components/chat/ConversationThread";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Button } from "@/components/ui/button";
import {
  getChatThreadIndicatorsKey,
  getChatTicketThreadStatusKey,
  useFetchOrCreateTicketThread,
  useFetchTicketThreadStatus,
} from "@/hooks/ChatHooks";

interface TicketThreadSectionProps {
  workspaceId: string;
  ticketId: string;
}

// "starting" is local UI state for the one-shot "Start chat" click; an existing thread loads on its own.
export const TicketThreadSection = ({ workspaceId, ticketId }: TicketThreadSectionProps) => {
  const [starting, setStarting] = useState(false);
  const client = useQueryClient();
  const { data: status, isPending: statusPending } = useFetchTicketThreadStatus(ticketId);
  const hasThread = status?.[ticketId] === true;
  const shouldLoad = hasThread || starting;
  const { data: conversation, error, isPending } = useFetchOrCreateTicketThread(workspaceId, ticketId, shouldLoad);

  useEffect(() => {
    if (!conversation) return;
    void client.invalidateQueries({ queryKey: [getChatThreadIndicatorsKey] });
    void client.invalidateQueries({ queryKey: [getChatTicketThreadStatusKey, ticketId] });
  }, [conversation, client, ticketId]);

  return (
    <div className="space-y-3 border-t border-border pt-6">
      <h2 className="text-sm font-semibold tracking-tight">Thread</h2>
      {statusPending && <LoadingDisplay label="Loading thread…" />}
      {!statusPending && !shouldLoad && (
        <Button variant="outline" size="sm" onClick={() => setStarting(true)}>
          <MessageSquare className="size-4" aria-hidden />
          Start chat
        </Button>
      )}
      {shouldLoad && isPending && <LoadingDisplay label="Loading thread…" />}
      {shouldLoad && error && <ErrorDisplay error={error} title="Failed to load the thread." />}
      {conversation && (
        <div className="h-96 overflow-hidden rounded-lg border border-border">
          <ConversationThread workspaceId={workspaceId} conversation={conversation} showHeader={false} />
        </div>
      )}
    </div>
  );
};
