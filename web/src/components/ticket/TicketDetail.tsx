import { Link } from "react-router";

import type { LiveSocket } from "@/api/ws";
import { DocBodyView } from "@/components/doc/DocBodyView";
import { DocPresenceBar } from "@/components/doc/DocPresenceBar";
import { CollabRichTextEditor } from "@/components/doc/collab/CollabRichTextEditor";
import { useCollabCommit } from "@/components/doc/collab/useCollabCommit";
import { useCollabSession } from "@/components/doc/collab/useCollabSession";
import { formatUpdatedAgo } from "@/components/doc/docTime";
import { TicketStatusBadge } from "@/components/ticket/TicketStatusBadge";
import { TitleTextarea } from "@/components/TitleTextarea";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useFetchMe } from "@/hooks/AuthHooks";
import { usePerson } from "@/hooks/PeopleHooks";
import { getTicketKey } from "@/hooks/TicketHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { personLabel } from "@/models/Person";
import type { Project } from "@/models/Project";
import { reporterLabel, type Ticket } from "@/models/Ticket";
import { effectiveAvatar } from "@/models/User";
import { useSessionStore } from "@/stores/sessionStore";

interface TicketDetailProps {
  ticket: Ticket;
  project?: Project;
  /** Test seam: the session's socket factory. */
  wsFactory?: (url: string) => LiveSocket;
}

// Mounted with key={ticket.id}: the title seeds once, so switching tickets must remount, not update in place.
// A writer edits the title and body live in the ticket's room, the way a doc is edited; a reader sees them static.
export const TicketDetail = ({ ticket, project, wsFactory }: TicketDetailProps) => {
  const token = useSessionStore((s) => s.token);
  const { data: me } = useFetchMe();
  const canEdit = useAreaAccess(ticket.project_id)?.("editTickets") ?? false;
  const avatar = me?.user ? effectiveAvatar(me.user) : "";
  // Joins once the viewer's name is known, so presence never shows a placeholder that reconnects a moment later.
  const session = useCollabSession(canEdit && me ? `tickets/${ticket.id}` : undefined, "edit", me?.user?.name ?? "", token, {
    reloadKey: [getTicketKey, ticket.id],
    ...(wsFactory ? { wsFactory } : {}),
    ...(avatar ? { avatar } : {}),
  });
  const { title, titleInputRef, onTitleChange, confirmTitle, onBodyChange } = useCollabCommit(session, ticket.title, ticket.body);
  const reporterPerson = usePerson(ticket.reporter.login ?? "");
  const wsPath = useWorkspacePath();
  const reporter = reporterLabel(ticket.reporter, () => personLabel(reporterPerson));

  return (
    <div className="space-y-6">
      <Link
        to={wsPath("/board")}
        className="inline-block font-mono text-xs text-muted-foreground transition-colors duration-150 ease-standard hover:text-foreground"
      >
        ← Board
      </Link>
      <div className="space-y-3">
        <div className="flex items-center gap-2">
          <span className="font-mono text-xs text-muted-foreground">
            {project ? `${project.prefix}-${ticket.number}` : ticket.id}
          </span>
          <TicketStatusBadge ticket={ticket} />
        </div>
        {session && (
          <h1>
            <TitleTextarea
              ref={titleInputRef}
              value={title}
              onValueChange={onTitleChange}
              onBlur={confirmTitle}
              blurOnEnter
              aria-label="Ticket title"
              className="text-3xl sm:text-4xl"
            />
          </h1>
        )}
        {!session && <h1 className="text-center text-3xl font-semibold tracking-tight sm:text-4xl">{ticket.title}</h1>}
        <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
          <p className="font-mono text-xs text-muted-foreground">
            created {formatUpdatedAgo(ticket.created_at)}
            {reporter && ` by ${reporter}`}
          </p>
          <DocPresenceBar participants={session?.participants ?? []} connected={session?.connected} updatedAt={ticket.updated_at} />
        </div>
      </div>
      <div className="rounded-2xl border border-border bg-card p-6 shadow-card sm:p-10">
        {session && (
          <CollabRichTextEditor
            session={session}
            value={ticket.body}
            onChange={onBodyChange}
            aria-label="Ticket description"
            attachTo={{ ticket_id: ticket.id }}
          />
        )}
        {!session && <DocBodyView body={ticket.body} />}
      </div>
    </div>
  );
};
