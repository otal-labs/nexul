import { useParams } from "react-router";

import { Container } from "@/components/Container";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { TicketPageBody } from "@/components/ticket/TicketPageBody";
import {
  useAddLabel,
  useFetchTicket,
  useRemoveLabel,
  useSetTicketAssignee,
  useSetTicketType,
  useUpdateTicket,
  useUpdateTicketStatus,
} from "@/hooks/TicketHooks";
import { useTicketPageResolution } from "@/hooks/useTicketPageResolution";
import { useWorkspaceStore } from "@/stores/workspaceStore";

interface TicketPageProps {
  /** Overrides the route param — used to embed a ticket's detail without navigating (e.g. Inbox's split view). */
  ticketId?: string;
}

export const TicketPage = ({ ticketId: ticketIdProp }: TicketPageProps = {}) => {
  const { ticketId: routeTicketId } = useParams<{ ticketId: string }>();
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { projects, ticketId, resolving, notFound } = useTicketPageResolution({
    ticketIdProp,
    routeTicketId,
  });
  const { data, error, isPending } = useFetchTicket(ticketId);
  const updateStatus = useUpdateTicketStatus();
  const updateTicket = useUpdateTicket();
  const setAssignee = useSetTicketAssignee();
  const setType = useSetTicketType();
  const addLabel = useAddLabel();
  const removeLabel = useRemoveLabel();

  const project = data && projects.find((p) => p.id === data.project_id);

  return (
    <Container className="p-6">
      {(isPending || resolving) && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {notFound && <ErrorDisplay title="Ticket not found" />}
      {data && (
        <TicketPageBody
          ticket={data}
          project={project || undefined}
          workspaceId={workspaceId}
          onSave={async (title, body) => {
            await updateTicket.mutateAsync({ id: data.id, title, body });
          }}
          onTransition={async (status) => {
            await updateStatus.mutateAsync({ id: data.id, status });
          }}
          onSetAssignee={async (id, assignee) => {
            await setAssignee.mutateAsync({ id, assignee });
          }}
          onSetType={async (id, typeId) => {
            await setType.mutateAsync({ id, typeId });
          }}
          onAddLabel={async (id, label) => {
            await addLabel.mutateAsync({ id, label });
          }}
          onRemoveLabel={async (id, label) => {
            await removeLabel.mutateAsync({ id, label });
          }}
        />
      )}
    </Container>
  );
};
