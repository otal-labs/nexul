import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useFetchTicketIdByKey } from "@/hooks/TicketHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { parseTicketKey } from "@/models/Ticket";

interface UseTicketPageResolutionArgs {
  ticketIdProp: string | undefined;
  routeTicketId: string | undefined;
}

// A PREFIX-NUMBER route param resolves to the real UUID through the server's key lookup; a UUID param passes straight through.
export const useTicketPageResolution = ({ ticketIdProp, routeTicketId }: UseTicketPageResolutionArgs) => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: projects = [] } = useFetchProjects();
  const key = !ticketIdProp && routeTicketId && parseTicketKey(routeTicketId) ? routeTicketId : undefined;
  const { data: keyTicketId, error: keyError } = useFetchTicketIdByKey(key, workspaceId);
  const ticketId = ticketIdProp ?? (key ? keyTicketId : routeTicketId);

  return { projects, ticketId, keyError };
};
