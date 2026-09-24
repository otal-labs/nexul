import { TicketThreadSection } from "@/components/chat/TicketThreadSection";
import { ReviewPanel } from "@/components/codereview/ReviewPanel";
import { PlaysBottomBar } from "@/components/play/PlaysBottomBar";
import { PlaysRailSection } from "@/components/play/PlaysRailSection";
import { TrailSection } from "@/components/play/TrailSection";
import { TicketDetail } from "@/components/ticket/TicketDetail";
import { TicketLinksSection } from "@/components/ticket/TicketLinksSection";
import { TicketPropertiesPanel } from "@/components/ticket/TicketPropertiesPanel";
import type { Project } from "@/models/Project";
import type { Ticket, TicketStatus as TicketStatusType } from "@/models/Ticket";

interface TicketPageBodyProps {
  ticket: Ticket;
  project: Project | undefined;
  workspaceId: string;
  onSave: (title: string, body: string) => Promise<void>;
  onTransition: (status: TicketStatusType) => Promise<void>;
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
      <TicketLinksSection ticket={ticket} />
      {workspaceId && <TrailSection workspaceId={workspaceId} targetType="ticket" targetId={ticket.id} />}
      {workspaceId && <TicketThreadSection workspaceId={workspaceId} ticketId={ticket.id} />}
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
