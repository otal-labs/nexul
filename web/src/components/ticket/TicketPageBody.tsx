import { TicketThreadSection } from "@/components/chat/TicketThreadSection";
import { PageTabs, PageTabsContent, type PageTab } from "@/components/PageTabs";
import { TicketDetail } from "@/components/ticket/TicketDetail";
import { ticketPageLayout } from "@/components/ticket/ticketPageLayout";
import { TicketRail } from "@/components/ticket/TicketRail";
import type { Project } from "@/models/Project";
import type { Ticket } from "@/models/Ticket";

interface TicketPageBodyProps {
  ticket: Ticket;
  project: Project | undefined;
  workspaceId: string;
  onTransition: (statusId: string) => Promise<void>;
  onSetType: (ticketId: string, typeId: string) => Promise<void>;
  onAddLabel: (ticketId: string, label: string) => Promise<void>;
  onRemoveLabel: (ticketId: string, label: string) => Promise<void>;
  /** Rendered inside another page (the Inbox split view), where the thread stays under the body. */
  embedded: boolean;
}

export const TicketPageBody = ({ ticket, project, workspaceId, embedded, ...handlers }: TicketPageBodyProps) => {
  const layout = ticketPageLayout(embedded);
  const hasThread = workspaceId !== "";
  const rail = <TicketRail ticket={ticket} workspaceId={workspaceId} {...handlers} />;
  const projectProp = project ? { project } : {};

  if (embedded) {
    return (
      <div className={layout.page}>
        <TicketDetail key={ticket.id} ticket={ticket} layout={layout} {...projectProp} />
        <div className={layout.rail}>{rail}</div>
        {hasThread && (
          <div className={layout.thread}>
            <TicketThreadSection workspaceId={workspaceId} ticketId={ticket.id} />
          </div>
        )}
      </div>
    );
  }

  const tabs: PageTab[] = [
    { value: "ticket", label: "Ticket" },
    { value: "thread", label: "Thread", hidden: !hasThread },
  ];
  return (
    <div className={layout.page} data-pane-layout="">
      <TicketDetail
        key={ticket.id}
        ticket={ticket}
        layout={layout}
        {...projectProp}
        frameBody={(article) => (
          <PageTabs label="Ticket sections" tabs={tabs} className={layout.tabs}>
            <PageTabsContent value="ticket" className={layout.ticketTab}>
              <div className={layout.ticketGrid}>
                {article}
                <div className={layout.rail}>{rail}</div>
              </div>
            </PageTabsContent>
            {hasThread && (
              <PageTabsContent value="thread" className={layout.thread}>
                <TicketThreadSection workspaceId={workspaceId} ticketId={ticket.id} pane />
              </PageTabsContent>
            )}
          </PageTabs>
        )}
      />
    </div>
  );
};
