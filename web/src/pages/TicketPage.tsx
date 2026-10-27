import { useParams } from "react-router";

import { Container } from "@/components/Container";
import { DetailErrorDisplay } from "@/components/DetailErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { TicketPageBody } from "@/components/ticket/TicketPageBody";
import { ticketPageLayout } from "@/components/ticket/ticketPageLayout";
import {
  useAddLabel,
  useFetchTicket,
  useRemoveLabel,
  useSetTicketType,
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
  const { projects, ticketId, keyError } = useTicketPageResolution({
    ticketIdProp,
    routeTicketId,
  });
  const { data, error: ticketError, isPending } = useFetchTicket(ticketId);
  const error = keyError ?? ticketError;
  const updateStatus = useUpdateTicketStatus();
  const setType = useSetTicketType();
  const addLabel = useAddLabel();
  const removeLabel = useRemoveLabel();

  const embedded = ticketIdProp !== undefined;
  const project = data && projects.find((p) => p.id === data.project_id);

  return (
    <Container className={ticketPageLayout(embedded).container}>
      {!error && isPending && <LoadingDisplay />}
      {error && <DetailErrorDisplay error={error} embedded={embedded} />}
      {data && (
        <TicketPageBody
          ticket={data}
          project={project || undefined}
          workspaceId={workspaceId}
          embedded={embedded}
          onTransition={async (status) => {
            await updateStatus.mutateAsync({ id: data.id, status });
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
