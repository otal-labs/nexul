import type { CSSProperties } from "react";

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
import { ThreadPaneResizeHandle } from "@/components/ticket/ThreadPaneResizeHandle";
import { ThreadVariantSwitcher } from "@/components/ticket/ThreadVariantSwitcher";
import { useThreadVariant } from "@/components/ticket/threadVariants";
import type { Project } from "@/models/Project";
import type { Ticket } from "@/models/Ticket";
import { useThreadPaneStore } from "@/stores/threadPaneStore";

interface TicketPageBodyProps {
  ticket: Ticket;
  project: Project | undefined;
  workspaceId: string;
  onSave: (title: string, body: string) => Promise<void>;
  onTransition: (statusId: string) => Promise<void>;
  onSetType: (ticketId: string, typeId: string) => Promise<void>;
  onAddLabel: (ticketId: string, label: string) => Promise<void>;
  onRemoveLabel: (ticketId: string, label: string) => Promise<void>;
  embedded: boolean;
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
  embedded,
}: TicketPageBodyProps) => {
  const variant = useThreadVariant(embedded);
  const threadWidth = useThreadPaneStore((s) => s.width);
  const gridStyle = variant.wide && threadWidth !== null ? { "--thread-pane-width": `${threadWidth}px` } : undefined;
  return (
    <div className={variant.grid} data-thread-grid="" style={gridStyle as CSSProperties | undefined}>
      {!embedded && <ThreadVariantSwitcher current={variant} />}
      <div className={variant.body}>
        <TicketDetail key={ticket.id} ticket={ticket} {...(project ? { project } : {})} onSave={onSave} />
      </div>
      {workspaceId !== "" && (
        <div className={variant.thread}>
          <TicketThreadSection workspaceId={workspaceId} ticketId={ticket.id} variant={variant} />
          {variant.wide && <ThreadPaneResizeHandle />}
        </div>
      )}
      <div className={variant.rail}>
        <TicketPropertiesPanel
          ticket={ticket}
          onTransition={onTransition}
          onSetType={onSetType}
          onAddLabel={onAddLabel}
          onRemoveLabel={onRemoveLabel}
        >
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
          <PlaysRailSection ticket={ticket} />
        </TicketPropertiesPanel>
      </div>
    </div>
  );
};
