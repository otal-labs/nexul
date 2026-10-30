import { useId } from "react";
import { Link } from "react-router";

import { microheaderClass } from "@/components/Microheader";
import { ThreadPreviewRow } from "@/components/ticket/ThreadPreviewRow";
import { useFetchMessages, useFetchOrCreateTicketThread, useFetchTicketThreadStatus } from "@/hooks/ChatHooks";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import type { Message } from "@/models/Chat";

interface TicketThreadPreviewProps {
  workspaceId: string;
  ticketId: string;
  threadPath: string;
}

const shown = 3;

// The thread's own last lines, newest last; a note's file says nothing in one line, so notes are left out.
const latestLines = (messages: Message[]): Message[] =>
  messages
    .filter((m) => !m.deleted_at && !m.attachment_id && m.body.trim() !== "")
    .sort((a, b) => a.created_at.localeCompare(b.created_at))
    .slice(-shown);

// The thread's last few lines under the spec, so the body and the talk about it share a screen; the tab holds the rest.
// It reads the messages the Thread tab loads, and never opens a thread that doesn't exist yet.
export const TicketThreadPreview = ({ workspaceId, ticketId, threadPath }: TicketThreadPreviewProps) => {
  const headingId = useId();
  const { data: status } = useFetchTicketThreadStatus(ticketId);
  const { data: conversation } = useFetchOrCreateTicketThread(workspaceId, ticketId, status?.[ticketId] === true);
  const { data: messages } = useFetchMessages(conversation?.id);
  const resolvePerson = usePersonLookup(workspaceId);
  const latest = messages ? latestLines(messages) : [];

  return (
    latest.length > 0 && (
      <section aria-labelledby={headingId} className="mt-10 max-w-[68ch] space-y-2 border-t border-border pt-6">
        <div className="flex items-center justify-between">
          <h2 id={headingId} className={microheaderClass}>
            Thread
          </h2>
          <Link to={threadPath} className="text-xs text-muted-foreground underline-offset-4 hover:text-foreground hover:underline">
            Open thread
          </Link>
        </div>
        <ul className="grid grid-cols-[auto_auto_auto_minmax(0,1fr)] gap-x-2">
          {latest.map((message) => (
            <ThreadPreviewRow key={message.id} message={message} author={resolvePerson(message.author_id)} />
          ))}
        </ul>
      </section>
    )
  );
};
