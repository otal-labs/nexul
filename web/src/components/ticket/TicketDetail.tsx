import type { LiveSocket } from "@nexul/client-core/liveSocket";
import type { ReactNode } from "react";

import { CalendarTitleContext } from "@/components/chat/calendarTitle";
import { DocBodyView } from "@/components/doc/DocBodyView";
import { DocPresenceBar } from "@/components/doc/DocPresenceBar";
import { CollabRichTextEditor } from "@/components/doc/collab/CollabRichTextEditor";
import { useCollabCommit } from "@/components/doc/collab/useCollabCommit";
import { useCollabSession } from "@/components/doc/collab/useCollabSession";
import { ClampedTitle, PageHeader, pageTitleClassFor } from "@/components/PageHeader";
import { RowActionsMenu } from "@/components/settings/RowActionsMenu";
import { TicketHeaderStrip } from "@/components/ticket/TicketHeaderStrip";
import type { TicketPageLayout } from "@/components/ticket/ticketPageLayout";
import { TitleTextarea } from "@/components/TitleTextarea";
import { useActionState } from "@/hooks/useActionState";
import { useFetchMe } from "@/hooks/AuthHooks";
import { getTicketKey } from "@/hooks/TicketCache";
import { useTicketActions } from "@/hooks/useTicketActions";
import { useProjectCrumb, useWorkspaceCrumb } from "@/hooks/useCrumbs";
import { useEmbeddedCrumbs } from "@/hooks/useEmbeddedCrumbs";
import type { Project } from "@/models/Project";
import type { Ticket } from "@/models/Ticket";
import { effectiveAvatar } from "@/models/User";
import { useSessionStore } from "@/stores/sessionStore";
import { cn } from "@/lib/utils";

interface TicketDetailProps {
  ticket: Ticket;
  project?: Project;
  /** Where the header and the body sit in the ticket page's grid; they stack when omitted. */
  layout?: Pick<TicketPageLayout, "header" | "article">;
  /** Wraps the body under the header, so the ticket page can hold it in a tab. */
  frameBody?: (article: ReactNode) => ReactNode;
  /** Test seam: the session's socket factory. */
  wsFactory?: (url: string) => LiveSocket;
}

// Mounted with key={ticket.id}: the title seeds once, so switching tickets must remount, not update in place.
// A writer edits the title and body live in the ticket's room, the way a doc is edited; a reader sees them static.
export const TicketDetail = ({ ticket, project, layout, frameBody, wsFactory }: TicketDetailProps) => {
  const token = useSessionStore((s) => s.token);
  const { data: me } = useFetchMe();
  const canEdit = useActionState({ permission: "tickets:write", projectId: ticket.project_id }).state === "live";
  const avatar = me?.user ? effectiveAvatar(me.user) : "";
  // Joins once the viewer's name is known, so presence never shows a placeholder that reconnects a moment later.
  const session = useCollabSession(canEdit && me ? `tickets/${ticket.id}` : undefined, "edit", me?.user?.name ?? "", token, {
    reloadKey: [getTicketKey, ticket.id],
    ...(me?.user?.id ? { userId: me.user.id } : {}),
    ...(me?.user?.login ? { login: me.user.login } : {}),
    ...(wsFactory ? { wsFactory } : {}),
    ...(avatar ? { avatar } : {}),
  });
  const { title, titleInputRef, onTitleChange, confirmTitle, onBodyChange } = useCollabCommit(session, ticket.title, ticket.body);

  const ticketLabel = project ? `${project.prefix}-${ticket.number}` : ticket.id;
  const menu = useTicketActions(ticket, ticketLabel, project);

  const workspaceCrumb = useWorkspaceCrumb();
  const projectCrumb = useProjectCrumb(ticket.project_id);
  const crumbs = useEmbeddedCrumbs(projectCrumb ? [workspaceCrumb, projectCrumb, { ...projectCrumb, label: "Board" }] : [workspaceCrumb]);

  const article = (
    <article className={layout?.article}>
      {/* Open on the panel like a page, held to a reading measure; a card around it only framed empty space. */}
      <div className="max-w-[68ch]">
        <CalendarTitleContext value={ticket.title}>
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
        </CalendarTitleContext>
      </div>
    </article>
  );

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
        actions={<RowActionsMenu subject={ticketLabel} projectId={ticket.project_id} actions={menu} />}
        meta={
          <>
            <TicketHeaderStrip ticket={ticket} ticketLabel={ticketLabel} />
            <DocPresenceBar participants={session?.participants ?? []} connected={session?.connected} updatedAt={ticket.updated_at} />
          </>
        }
      />
      {frameBody && frameBody(article)}
      {!frameBody && article}
    </div>
  );
};
