import { useLocalSearchParams, useRouter } from "expo-router";
import { FlatList, View } from "react-native";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { PickerRow } from "@/components/PickerRow";
import { SheetTitle } from "@/components/SheetTitle";
import { statusStageDot } from "@/models/Status";
import { ticketKey } from "@/models/Ticket";
import { useFetchProject } from "@/hooks/ProjectHooks";
import { useFetchProjectStatuses } from "@/hooks/StatusHooks";
import { useFetchTicket, useUpdateTicketStatus } from "@/hooks/TicketHooks";
import { cn } from "@/lib/utils";

type StatusPickerParams = {
  ticketId: string;
  projectId: string;
  currentStatusId?: string;
};

export const StatusPickerScreen = () => {
  const router = useRouter();
  const { ticketId, projectId, currentStatusId } = useLocalSearchParams<StatusPickerParams>();
  const { data: statuses, error: statusesError, isPending } = useFetchProjectStatuses(projectId);
  const { data: ticket } = useFetchTicket(ticketId);
  const { data: project } = useFetchProject(projectId);
  const updateStatus = useUpdateTicketStatus();

  return (
    <View role="radiogroup" className="bg-popover pb-6">
      <SheetTitle title={ticket ? `Status · ${ticketKey(ticket, project?.prefix)}` : "Status"} />
      {isPending && <LoadingDisplay />}
      {statusesError && <ErrorDisplay error={statusesError} />}
      {updateStatus.error && <ErrorDisplay error={updateStatus.error} />}
      {statuses && (
        <FlatList
          data={statuses}
          keyExtractor={(s) => s.id}
          renderItem={({ item }) => (
            <PickerRow
              label={item.name}
              selected={item.id === currentStatusId}
              disabled={updateStatus.isPending}
              onPress={() => updateStatus.mutate({ id: ticketId, status: item.id }, { onSuccess: () => router.back() })}
              leading={<View className={cn("size-2.5 rounded-full", statusStageDot(item.kind))} />}
            />
          )}
        />
      )}
    </View>
  );
};
