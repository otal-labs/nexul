import { TicketThreadSection } from "@/components/chat/TicketThreadSection";
import { ReviewPanel } from "@/components/codereview/ReviewPanel";
import { PageTabs, PageTabsContent } from "@/components/PageTabs";
import { DecisionsCheckNotice } from "@/components/play/DecisionsCheckNotice";
import { PlaysBottomBar } from "@/components/play/PlaysBottomBar";
import { PlaysRailSection } from "@/components/play/PlaysRailSection";
import { TrailSection } from "@/components/play/TrailSection";
import { TicketDetail } from "@/components/ticket/TicketDetail";
import { TicketLinksSection } from "@/components/ticket/TicketLinksSection";
import { TicketPropertiesPanel } from "@/components/ticket/TicketPropertiesPanel";
import { TicketTestSection } from "@/components/ticket/TicketTestSection";
import type { Project } from "@/models/Project";
import type { Ticket } from "@/models/Ticket";

// The tab row already draws the hairline these sections open with when stacked.
const tabBodyClass = "[&>*:first-child]:border-t-0 [&>*:first-child]:pt-0";

interface TicketPageBodyProps {
  ticket: Ticket;
  project: Project | undefined;
  workspaceId: string;
  onSave: (title: string, body: string) => Promise<void>;
  onTransition: (statusId: string) => Promise<void>;
  onSetType: (ticketId: string, typeId: string) => Promise<void>;
  onAddLabel: (ticketId: string, label: string) => Promise<void>;
  onRemoveLabel: (ticketId: string, label: string) => Promise<void>;
}

export const TicketPageBody = ({
  ticket,
  project,
  workspaceId,
  onSave,
  onTransition,
  onSetType,
  onAddLabel,
  onRemoveLabel,
}: TicketPageBodyProps) => (
  <div className="grid gap-8 lg:grid-cols-[minmax(0,1fr)_18rem]">
    <div className="min-w-0 space-y-8">
      <TicketDetail key={ticket.id} ticket={ticket} {...(project ? { project } : {})} onSave={onSave} />
      <PageTabs
        label="Ticket sections"
        tabs={[
          { value: "thread", label: "Thread", hidden: !workspaceId },
          { value: "testing", label: "Testing" },
          { value: "links", label: "Links" },
          { value: "activity", label: "Activity", hidden: !workspaceId },
        ]}
      >
        <PageTabsContent value="thread" className={tabBodyClass}>
          <TicketThreadSection workspaceId={workspaceId} ticketId={ticket.id} />
        </PageTabsContent>
        <PageTabsContent value="testing" className={tabBodyClass}>
          <TicketTestSection ticket={ticket} />
        </PageTabsContent>
        <PageTabsContent value="links" className={tabBodyClass}>
          <TicketLinksSection ticket={ticket} />
          <DecisionsCheckNotice ticketId={ticket.id} />
        </PageTabsContent>
        <PageTabsContent value="activity" className={tabBodyClass}>
          <TrailSection
            workspaceId={workspaceId}
            targetType="ticket"
            targetId={ticket.id}
            emptyMessage="No plays have run on this ticket yet."
          />
        </PageTabsContent>
      </PageTabs>
    </div>
    <TicketPropertiesPanel
      ticket={ticket}
      onTransition={onTransition}
      onSetType={onSetType}
      onAddLabel={onAddLabel}
      onRemoveLabel={onRemoveLabel}
    >
      <PlaysRailSection ticket={ticket} />
      <ReviewPanel ticketId={ticket.id} />
    </TicketPropertiesPanel>
    <PlaysBottomBar ticket={ticket} />
  </div>
);
