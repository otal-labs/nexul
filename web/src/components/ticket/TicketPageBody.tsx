import { useId, useState } from "react";

import { TicketThreadSection } from "@/components/chat/TicketThreadSection";
import { TicketNotesTab } from "@/components/note/TicketNotesTab";
import { PageTabs, PageTabsContent, type PageTab } from "@/components/PageTabs";
import { TicketDetail } from "@/components/ticket/TicketDetail";
import { TicketDetailsToggle } from "@/components/ticket/TicketDetailsToggle";
import { ticketPageLayout } from "@/components/ticket/ticketPageLayout";
import { TicketRail } from "@/components/ticket/TicketRail";
import { TicketThreadPreview } from "@/components/ticket/TicketThreadPreview";
import { useFetchTicketNotes } from "@/hooks/NoteHooks";
import { useTicketRunState } from "@/hooks/TrailHooks";
import { useTabPath } from "@/hooks/useTabPath";
import type { Project } from "@/models/Project";
import type { Ticket } from "@/models/Ticket";
import { cn } from "@/lib/utils";

interface TicketPageBodyProps {
  ticket: Ticket;
  project: Project | undefined;
  workspaceId: string;
  onTransition: (statusId: string) => Promise<void>;
  onSetType: (ticketId: string, typeId: string) => Promise<void>;
  onAddLabel: (ticketId: string, label: string) => Promise<void>;
  onRemoveLabel: (ticketId: string, label: string) => Promise<void>;
  /** Rendered inside another page (the Inbox split view), whose detail pane is already the panel. */
  embedded: boolean;
}

export const TicketPageBody = ({ ticket, project, workspaceId, embedded, ...handlers }: TicketPageBodyProps) => {
  const layout = ticketPageLayout(embedded);
  const hasThread = workspaceId !== "";
  const projectProp = project ? { project } : {};
  const { data: notes } = useFetchTicketNotes(ticket.id);
  const hasNotes = !!notes && notes.notes.length > 0;
  const waiting = useTicketRunState(ticket.project_id, ticket.id) === "waiting";
  const { current, tabPath } = useTabPath();
  const onNotes = hasNotes && current === "notes";
  const [detailsOpen, setDetailsOpen] = useState(false);
  const detailsId = useId();

  const tabs: PageTab[] = [
    { value: "ticket", label: "Ticket" },
    { value: "thread", label: "Thread", hidden: !hasThread, dot: waiting ? "needs your answer" : undefined },
    { value: "notes", label: "Notes", hidden: !hasNotes },
  ];
  return (
    <div className={layout.page} data-pane-layout={embedded ? undefined : ""}>
      <TicketDetail
        key={ticket.id}
        ticket={ticket}
        layout={layout}
        {...projectProp}
        frameBody={(article) => (
          <div className={layout.body}>
            {!onNotes && <TicketDetailsToggle open={detailsOpen} onOpenChange={setDetailsOpen} controls={detailsId} className={layout.detailsToggle} />}
            {!onNotes && (
              <div id={detailsId} className={cn(layout.rail, detailsOpen && layout.railOpen)}>
                <div className={layout.railScroll}>
                  <TicketRail ticket={ticket} workspaceId={workspaceId} {...handlers} />
                </div>
              </div>
            )}
            <PageTabs label="Ticket sections" tabs={tabs} className={cn(layout.tabs, onNotes && layout.tabsWide)}>
              <PageTabsContent value="ticket" className={layout.ticketTab}>
                {article}
                {hasThread && <TicketThreadPreview workspaceId={workspaceId} ticketId={ticket.id} threadPath={tabPath("thread")} />}
              </PageTabsContent>
              {hasThread && (
                <PageTabsContent value="thread" className={layout.thread}>
                  <TicketThreadSection workspaceId={workspaceId} ticketId={ticket.id} pane />
                </PageTabsContent>
              )}
              {notes && hasNotes && (
                <PageTabsContent value="notes" className={layout.ticketTab}>
                  <TicketNotesTab ticketId={ticket.id} notes={notes} gridClassName={layout.notesGrid} railClassName={layout.notesRail} />
                </PageTabsContent>
              )}
            </PageTabs>
          </div>
        )}
      />
    </div>
  );
};
