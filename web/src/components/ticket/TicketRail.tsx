import { AttachmentsSection } from "@/components/attachment/AttachmentsSection";
import { ReviewPanel } from "@/components/codereview/ReviewPanel";
import { DecisionsCheckNotice } from "@/components/play/DecisionsCheckNotice";
import { PlaysRailSection } from "@/components/play/PlaysRailSection";
import { TrailSection } from "@/components/play/TrailSection";
import { DevelopmentSection } from "@/components/ticket/DevelopmentSection";
import { TicketBugsSection } from "@/components/ticket/TicketBugsSection";
import { TicketCreatedLine } from "@/components/ticket/TicketCreatedLine";
import { TicketLinksSection } from "@/components/ticket/TicketLinksSection";
import { TicketPropertiesPanel } from "@/components/ticket/TicketPropertiesPanel";
import { TicketTestSection } from "@/components/ticket/TicketTestSection";
import type { Ticket } from "@/models/Ticket";

interface TicketRailProps {
  ticket: Ticket;
  workspaceId: string;
  onTransition: (statusId: string) => Promise<void>;
  onSetType: (ticketId: string, typeId: string) => Promise<void>;
  onAddLabel: (ticketId: string, label: string) => Promise<void>;
  onRemoveLabel: (ticketId: string, label: string) => Promise<void>;
}

export const TicketRail = ({ ticket, workspaceId, onTransition, onSetType, onAddLabel, onRemoveLabel }: TicketRailProps) => (
  <TicketPropertiesPanel
    ticket={ticket}
    onTransition={onTransition}
    onSetType={onSetType}
    onAddLabel={onAddLabel}
    onRemoveLabel={onRemoveLabel}
  >
    <TicketLinksSection ticket={ticket} />
    <DevelopmentSection ticketId={ticket.id} />
    <ReviewPanel ticketId={ticket.id} />
    <TicketTestSection ticket={ticket} />
    <PlaysRailSection ticket={ticket} />
    <DecisionsCheckNotice ticketId={ticket.id} />
    <AttachmentsSection owner={{ ticket_id: ticket.id }} projectId={ticket.project_id} className="px-2 [&_[role=status]]:text-xs" actionPlacement="end" />
    <TicketBugsSection ticket={ticket} />
    {workspaceId !== "" && (
      <TrailSection
        workspaceId={workspaceId}
        targetType="ticket"
        targetId={ticket.id}
        emptyMessage="No play runs yet."
        className="border-t-0 pt-0"
      />
    )}
    <TicketCreatedLine ticket={ticket} />
  </TicketPropertiesPanel>
);
