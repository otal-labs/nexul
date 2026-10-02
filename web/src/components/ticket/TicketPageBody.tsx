import { AttachmentsSection } from "@/components/attachment/AttachmentsSection";
import { TicketThreadSection } from "@/components/chat/TicketThreadSection";
import { ReviewPanel } from "@/components/codereview/ReviewPanel";
import { DecisionsCheckNotice } from "@/components/play/DecisionsCheckNotice";
import { PlaysRailSection } from "@/components/play/PlaysRailSection";
import { TrailSection } from "@/components/play/TrailSection";
import { DevelopmentSection } from "@/components/ticket/DevelopmentSection";
import { TicketBugsSection } from "@/components/ticket/TicketBugsSection";
import { TicketDetail } from "@/components/ticket/TicketDetail";
import { TicketLinksSection } from "@/components/ticket/TicketLinksSection";
import { TicketPropertiesPanel } from "@/components/ticket/TicketPropertiesPanel";
import { TicketTestSection } from "@/components/ticket/TicketTestSection";
import type { Project } from "@/models/Project";
import type { Ticket } from "@/models/Ticket";

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
      {workspaceId !== "" && <TicketThreadSection workspaceId={workspaceId} ticketId={ticket.id} />}
    </div>
    <TicketPropertiesPanel
      ticket={ticket}
      onTransition={onTransition}
      onSetType={onSetType}
      onAddLabel={onAddLabel}
      onRemoveLabel={onRemoveLabel}
    >
      <PlaysRailSection ticket={ticket} />
      <DevelopmentSection ticketId={ticket.id} />
      <ReviewPanel ticketId={ticket.id} />
      <AttachmentsSection owner={{ ticket_id: ticket.id }} className="px-2" actionPlacement="end" />
      <TicketLinksSection ticket={ticket} />
      <DecisionsCheckNotice ticketId={ticket.id} />
      <TicketTestSection ticket={ticket} />
      <TicketBugsSection ticket={ticket} />
      {workspaceId !== "" && (
        <TrailSection
          workspaceId={workspaceId}
          targetType="ticket"
          targetId={ticket.id}
          emptyMessage="No plays have run on this ticket yet."
          className="border-t-0 pt-0"
        />
      )}
    </TicketPropertiesPanel>
  </div>
);
