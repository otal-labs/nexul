import { ticketsOnYou } from "@/components/home/homeTickets";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useFetchProjectStatuses } from "@/hooks/StatusHooks";
import { useFetchTicketsByProject } from "@/hooks/TicketHooks";

// Read from the board's own list, so the board opened after the home (or the home after the board) loads nothing new.
export const useTicketsOnYou = (projectId: string) => {
  const tickets = useFetchTicketsByProject(projectId);
  const statuses = useFetchProjectStatuses(projectId);
  const me = useFetchMe();
  const error = tickets.error ?? statuses.error ?? me.error;
  const data = tickets.data && statuses.data && me.data ? ticketsOnYou(tickets.data, statuses.data, me.data.user) : undefined;
  return { data, error, isPending: !error && !data };
};
