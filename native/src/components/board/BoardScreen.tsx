import { useRouter } from "expo-router";
import { SquareKanban } from "lucide-react-native";
import { useState } from "react";
import { View } from "react-native";
import { ScrollView } from "react-native-gesture-handler";

import { BoardDragContext, useBoardDragState } from "@/components/board/boardDrag";
import { BoardDragLayer } from "@/components/board/BoardDragLayer";
import { BoardHeader } from "@/components/board/BoardHeader";
import { BoardStageSummary } from "@/components/board/BoardStageSummary";
import { MineToggle } from "@/components/board/MineToggle";
import { StatusSection } from "@/components/board/StatusSection";
import { EmptyState } from "@/components/EmptyState";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { HandOff, useLoaderShown } from "@/components/HandOff";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ProjectRevokedState } from "@/components/project/ProjectRevokedState";
import { FieldScreen } from "@/components/FieldScreen";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useFetchProjects, useRevokedProject } from "@/hooks/ProjectHooks";
import { useFetchProjectStatuses } from "@/hooks/StatusHooks";
import { useFetchTicketsByProject, useUpdateTicketStatus } from "@/hooks/TicketHooks";
import { useFetchProjectTicketTypes } from "@/hooks/TicketTypeHooks";
import { useAreaAccess } from "@/hooks/WorkspaceHooks";
import { effectiveProject } from "@/models/Project";
import { useBoardStore } from "@/stores/boardStore";

export const BoardScreen = () => {
  const router = useRouter();
  const [mineOnly, setMineOnly] = useState(false);
  const { data: me } = useFetchMe(true);
  const { data: projects, error: projectsError, isPending: projectsPending } = useFetchProjects();
  const selectedProjectId = useBoardStore((s) => s.selectedProjectId);
  // A picked project taken away while open stays revoked instead of the board quietly showing another one.
  const revoked = useRevokedProject(selectedProjectId ?? undefined);
  const project = revoked ? undefined : effectiveProject(projects, selectedProjectId);
  const canReadTickets = useAreaAccess(project?.id)?.("tickets");
  const boardId = canReadTickets ? project?.id : undefined;
  const { data: statuses, error: statusesError, isPending: statusesPending } = useFetchProjectStatuses(boardId);
  const { data: tickets, error: ticketsError, isPending: ticketsPending } = useFetchTicketsByProject(boardId);
  const { data: ticketTypes } = useFetchProjectTicketTypes(boardId);

  const isPending = projectsPending || (!!boardId && (statusesPending || ticketsPending));
  const waited = useLoaderShown(isPending);
  const error = projectsError ?? statusesError ?? ticketsError;
  const filteredTickets =
    mineOnly && me ? tickets?.filter((t) => t.developer === me.user.login || t.tester === me.user.login) : tickets;
  const board = project && boardId && statuses && filteredTickets;
  const updateStatus = useUpdateTicketStatus();
  const drag = useBoardDragState(statuses ?? [], (id, status) => updateStatus.mutate({ id, status }));

  return (
    <FieldScreen>
      <BoardDragContext value={drag}>
        <ScrollView className="flex-1" contentContainerClassName="pb-6" scrollEnabled={!drag.lifted}>
          <BoardHeader project={revoked ? undefined : project} ticketCount={tickets?.length} />
          {isPending && <LoadingDisplay message="Loading the board" />}
          {error && <ErrorDisplay error={error} />}
          {updateStatus.error && <ErrorDisplay error={updateStatus.error} />}
          {revoked && <ProjectRevokedState />}
          {project && canReadTickets === false && <EmptyState title="This page doesn't exist" message="You can't see this project's board." />}
          {!revoked && projects && projects.length === 0 && (
            <EmptyState icon={SquareKanban} title="No projects yet" message="A project and its board are created on the web." />
          )}
          {board && tickets && (
            <HandOff after={waited}>
              <BoardStageSummary statuses={statuses} tickets={tickets} />
              <View className="px-5 pb-3 pt-2">
                <MineToggle on={mineOnly} onToggle={() => setMineOnly((v) => !v)} />
              </View>
              {statuses.map((status) => (
                <StatusSection
                  key={status.id}
                  status={status}
                  tickets={filteredTickets.filter((t) => t.status === status.id)}
                  projectPrefix={project.prefix}
                  ticketTypes={ticketTypes}
                  onOpenTicket={(id) => router.push(`/board/ticket/${id}`)}
                />
              ))}
            </HandOff>
          )}
        </ScrollView>
        <BoardDragLayer projectPrefix={project?.prefix} ticketTypes={ticketTypes} />
      </BoardDragContext>
    </FieldScreen>
  );
};
