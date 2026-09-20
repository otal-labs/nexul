import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useFetchTicketsByProject } from "@/hooks/TicketHooks";
import { resolveProject } from "@/models/Project";
import { parseTicketKey, type Ticket } from "@/models/Ticket";

const pickKeyTicketId = (
  key: ReturnType<typeof parseTicketKey>,
  keyTickets: Ticket[] | undefined,
  routeTicketId: string | undefined,
) => (key ? keyTickets?.find((t) => t.number === key.number)?.id : routeTicketId);

interface UseTicketPageResolutionArgs {
  ticketIdProp: string | undefined;
  routeTicketId: string | undefined;
}

// A PREFIX-NUMBER route param resolves client-side to the real UUID; a UUID param passes straight through.
export const useTicketPageResolution = ({ ticketIdProp, routeTicketId }: UseTicketPageResolutionArgs) => {
  const { data: projects = [], isPending: projectsPending } = useFetchProjects();
  const key = !ticketIdProp && routeTicketId ? parseTicketKey(routeTicketId) : undefined;
  const keyProject = key && resolveProject(projects, key.prefix);
  const { data: keyTickets, isPending: keyTicketsPending } = useFetchTicketsByProject(keyProject?.id);
  const ticketId = ticketIdProp ?? pickKeyTicketId(key, keyTickets, routeTicketId);
  const stillResolvingKey = Boolean(key) && !ticketId;
  const resolving = stillResolvingKey && (projectsPending || keyTicketsPending);
  const notFound = stillResolvingKey && !resolving;

  return { projects, ticketId, resolving, notFound };
};
