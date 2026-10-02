import { useParams } from "react-router";

import { Container } from "@/components/Container";
import { DetailErrorDisplay } from "@/components/DetailErrorDisplay";
import { ErrorScreen } from "@/components/ErrorScreen";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { TicketPageBody } from "@/components/ticket/TicketPageBody";
import {
  useAddLabel,
  useFetchTicket,
  useRemoveLabel,
  useSetTicketType,
  useUpdateTicket,
  useUpdateTicketStatus,
} from "@/hooks/TicketHooks";
import { useThreadVariant } from "@/components/ticket/threadVariants";
import { useTicketPageResolution } from "@/hooks/useTicketPageResolution";
import { cn } from "@/lib/utils";
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
  const setType = useSetTicketType();
  const addLabel = useAddLabel();
  const removeLabel = useRemoveLabel();
  const wide = useThreadVariant(ticketIdProp !== undefined).wide === true;

  const project = data && projects.find((p) => p.id === data.project_id);

  return (
    <Container className={cn("p-6", wide && "@container max-w-none")}>
      {(isPending || resolving) && <LoadingDisplay />}
      {error && <DetailErrorDisplay error={error} embedded={ticketIdProp !== undefined} />}
      {notFound && <ErrorScreen />}
      {data && (
        <TicketPageBody
          ticket={data}
          project={project || undefined}
          workspaceId={workspaceId}
          embedded={ticketIdProp !== undefined}
          onSave={async (title, body) => {
            await updateTicket.mutateAsync({ id: data.id, title, body });
          }}
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
