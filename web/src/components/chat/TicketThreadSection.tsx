import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";

import { cn } from "@/lib/utils";
import { ConversationThread } from "@/components/chat/ConversationThread";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { ThreadStartPrompt } from "@/components/chat/ThreadStartPrompt";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { STACKED, type ThreadVariant } from "@/components/ticket/threadVariants";
import {
  getChatThreadIndicatorsKey,
  getChatTicketThreadStatusKey,
  useFetchOrCreateTicketThread,
  useFetchTicketThreadStatus,
} from "@/hooks/ChatHooks";

interface TicketThreadSectionProps {
  workspaceId: string;
  ticketId: string;
  variant?: ThreadVariant;
}

// "starting" is local UI state for the one-shot "Start chat" click; an existing thread loads on its own.
export const TicketThreadSection = ({ workspaceId, ticketId, variant = STACKED }: TicketThreadSectionProps) => {
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
    <div className={variant.frame}>
      <h2 className="shrink-0 text-sm font-semibold tracking-tight">Thread</h2>
      {statusPending && <LoadingDisplay label="Loading thread…" />}
      {!statusPending && !shouldLoad && (
        <ThreadStartPrompt empty={variant.empty} onStart={() => setStarting(true)} />
      )}
      {shouldLoad && isPending && <LoadingDisplay label="Loading thread…" />}
      {shouldLoad && error && <ErrorDisplay error={error} title="Failed to load the thread." />}
      {conversation && (
        <div className={cn("overflow-hidden rounded-lg border border-border", variant.box)}>
          <ConversationThread
            workspaceId={workspaceId}
            conversation={conversation}
            showHeader={false}
            composerTop={variant.composerTop}
          />
        </div>
      )}
    </div>
  );
};
