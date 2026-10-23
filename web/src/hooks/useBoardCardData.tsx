import { createContext, useContext } from "react";

import type { Person } from "@nexul/client-core/person";

import type { ActiveTargets } from "@/hooks/TrailHooks";
import type { BoardStatus } from "@/models/Status";
import type { BlockersByTicket } from "@/models/TicketLink";
import type { TicketType } from "@/models/TicketType";

// What every card on a board reads, subscribed once by BoardCardProvider instead of once per card.
export interface BoardCardData {
  prefix: string;
  ticketTypes: TicketType[] | undefined;
  labelColors: Record<string, string> | undefined;
  threadIndicators: Record<string, boolean> | undefined;
  statuses: BoardStatus[] | undefined;
  blockers: BlockersByTicket | undefined;
  activeTrails: ActiveTargets | undefined;
  people: Person[] | undefined;
}

export const BoardCardDataContext = createContext<BoardCardData | null>(null);
// Apart from the data so a live update to it never re-renders every card's drag wrapper.
export const CanMoveTicketsContext = createContext(false);

export const useBoardCardData = (): BoardCardData => {
  const data = useContext(BoardCardDataContext);
  if (!data) throw new Error("useBoardCardData must be used inside a BoardCardProvider");
  return data;
};

export const useCanMoveTickets = () => useContext(CanMoveTicketsContext);
