import { LegendList } from "@legendapp/list/react-native";
import { useRouter } from "expo-router";
import SquareKanban from "lucide-react-native/icons/square-kanban";
import { useCallback, useMemo, useState } from "react";
import { View, type ScrollViewProps } from "react-native";
import { ScrollView } from "react-native-gesture-handler";

import { namesAccount, ticketDeveloper, ticketTester } from "@nexul/client-core/ticketPeople";

import { BoardDragContext, useBoardDragState, useLifted } from "@/components/board/boardDrag";
import { BoardDragLayer } from "@/components/board/BoardDragLayer";
import { BoardHeader } from "@/components/board/BoardHeader";
import { BoardLabelColorsContext } from "@/components/board/boardLabelColors";
import { BoardRowView, buildBoardRows } from "@/components/board/BoardRows";
import { BoardStageSummary } from "@/components/board/BoardStageSummary";
import { MineToggle } from "@/components/board/MineToggle";
import { EmptyState } from "@/components/EmptyState";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { HandOff, useLoaderShown } from "@/components/HandOff";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ProjectRevokedState } from "@/components/project/ProjectRevokedState";
import { FieldScreen } from "@/components/FieldScreen";
import { useActionState } from "@/hooks/useActionState";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useBoardLabelColors } from "@/hooks/LabelHooks";
import { useFetchProjects, useRevokedProject } from "@/hooks/ProjectHooks";
import { useFetchProjectStatuses } from "@/hooks/StatusHooks";
import { useFetchTicketsByProject, useUpdateTicketStatus } from "@/hooks/TicketHooks";
import { useFetchProjectTicketTypes } from "@/hooks/TicketTypeHooks";
import { useAreaAccess } from "@/hooks/WorkspaceHooks";
import { effectiveProject } from "@/models/Project";
import { useBoardStore } from "@/stores/boardStore";

// The cards' long-press drag only coexists with the scroll through gesture-handler's own scroll view.
const GestureScrollView = (props: ScrollViewProps) => <ScrollView {...props} />;

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
  const filteredTickets = useMemo(
    () =>
      mineOnly && me ? tickets?.filter((t) => namesAccount(ticketDeveloper(t), me.user) || namesAccount(ticketTester(t), me.user)) : tickets,
    [mineOnly, me, tickets],
  );
  const board = project && boardId && statuses && filteredTickets;
  const { data: labelColors } = useBoardLabelColors(boardId);
  const rows = useMemo(() => (board ? buildBoardRows(statuses, filteredTickets, ticketTypes) : []), [board, statuses, filteredTickets, ticketTypes]);
  const openTicket = useCallback((id: string) => router.push(`/board/ticket/${id}`), [router]);
  // mutate is stable, unlike the mutation's result object, so the drag context keeps its identity across renders.
  const { mutate: updateStatus, error: updateError } = useUpdateTicketStatus();
  const moveTicket = useCallback((id: string, status: string) => updateStatus({ id, status }), [updateStatus]);
  const drag = useBoardDragState(statuses ?? [], moveTicket);
  const holding = useLifted(drag, (held) => !!held);
  // Without tickets:write a card can't be held or moved, so the drag context is simply absent.
  const canMove = useActionState({ permission: "tickets:write", projectId: project?.id }).state === "live";

  return (
    <FieldScreen>
      <BoardDragContext value={canMove ? drag : null}>
        <BoardLabelColorsContext value={labelColors}>
          <LegendList
            className="flex-1"
            data={rows}
            keyExtractor={(row) => row.key}
            renderItem={({ item }) => <BoardRowView row={item} projectPrefix={project?.prefix} onOpenTicket={openTicket} />}
            renderScrollComponent={GestureScrollView}
            estimatedItemSize={96}
            recycleItems={false}
            contentContainerClassName="pb-6"
            scrollEnabled={!holding}
            ListHeaderComponent={
              <>
                <BoardHeader project={revoked ? undefined : project} ticketCount={tickets?.length} />
                {isPending && <LoadingDisplay message="Loading the board" />}
                {error && <ErrorDisplay error={error} />}
                {updateError && <ErrorDisplay error={updateError} />}
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
                  </HandOff>
                )}
              </>
            }
          />
          <BoardDragLayer projectPrefix={project?.prefix} ticketTypes={ticketTypes} />
        </BoardLabelColorsContext>
      </BoardDragContext>
    </FieldScreen>
  );
};
