import type { LiveSocket } from "@/api/ws";
import { DocBodyView } from "@/components/doc/DocBodyView";
import { DocPresenceBar } from "@/components/doc/DocPresenceBar";
import { CollabRichTextEditor } from "@/components/doc/collab/CollabRichTextEditor";
import { useCollabCommit } from "@/components/doc/collab/useCollabCommit";
import { useCollabSession } from "@/components/doc/collab/useCollabSession";
import { formatUpdatedAgo } from "@/components/doc/docTime";
import { ClampedTitle, PageHeader, pageTitleClassFor } from "@/components/PageHeader";
import { TicketStatusBadge } from "@/components/ticket/TicketStatusBadge";
import type { TicketPageLayout } from "@/components/ticket/ticketPageLayout";
import { TitleTextarea } from "@/components/TitleTextarea";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useFetchMe } from "@/hooks/AuthHooks";
import { usePerson } from "@/hooks/PeopleHooks";
import { getTicketKey } from "@/hooks/TicketHooks";
import { useProjectCrumb, useWorkspaceCrumb } from "@/hooks/useCrumbs";
import { useEmbeddedCrumbs } from "@/hooks/useEmbeddedCrumbs";
import { personLabel } from "@/models/Person";
import type { Project } from "@/models/Project";
import { reporterLabel, type Ticket } from "@/models/Ticket";
import { effectiveAvatar } from "@/models/User";
import { useSessionStore } from "@/stores/sessionStore";
import { cn } from "@/lib/utils";

interface TicketDetailProps {
  ticket: Ticket;
  project?: Project;
  /** Where the header and the body sit in the ticket page's grid; they stack when omitted. */
  layout?: Pick<TicketPageLayout, "header" | "article">;
  /** Test seam: the session's socket factory. */
  wsFactory?: (url: string) => LiveSocket;
}

// Mounted with key={ticket.id}: the title seeds once, so switching tickets must remount, not update in place.
// A writer edits the title and body live in the ticket's room, the way a doc is edited; a reader sees them static.
export const TicketDetail = ({ ticket, project, layout, wsFactory }: TicketDetailProps) => {
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
  const reporter = reporterLabel(ticket.reporter, () => personLabel(reporterPerson));

  const workspaceCrumb = useWorkspaceCrumb();
  const projectCrumb = useProjectCrumb(ticket.project_id);
  const crumbs = useEmbeddedCrumbs(projectCrumb ? [workspaceCrumb, projectCrumb, { ...projectCrumb, label: "Board" }] : [workspaceCrumb]);

  return (
    <div className={cn(!layout && "space-y-6", layout && "contents")}>
      <PageHeader
        className={layout?.header ?? ""}
        crumbs={crumbs}
        title={
          <ClampedTitle title={title}>
            {session && (
              <h1>
                <TitleTextarea
                  ref={titleInputRef}
                  value={title}
                  onValueChange={onTitleChange}
                  onBlur={confirmTitle}
                  blurOnEnter
                  aria-label="Ticket title"
                  className={pageTitleClassFor(title)}
                />
              </h1>
            )}
            {!session && (
              <h1 dir="auto" className={pageTitleClassFor(ticket.title)}>
                {ticket.title}
              </h1>
            )}
          </ClampedTitle>
        }
        meta={
          <>
            <span className="rounded-md bg-muted px-1.5 py-0.5 font-mono text-xs whitespace-nowrap text-foreground">
              {project ? `${project.prefix}-${ticket.number}` : ticket.id}
            </span>
            <TicketStatusBadge ticket={ticket} />
            <span className="font-mono text-xs">
              created {formatUpdatedAgo(ticket.created_at)}
              {reporter && ` by ${reporter}`}
            </span>
            <DocPresenceBar participants={session?.participants ?? []} connected={session?.connected} updatedAt={ticket.updated_at} />
          </>
        }
      />
      <article className={layout?.article}>
        {/* Open on the panel like a page, held to a reading measure; a card around it only framed empty space. */}
        <div className="max-w-[68ch]">
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
      </article>
    </div>
  );
};
