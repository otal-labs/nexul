import { useLocalSearchParams, useRouter } from "expo-router";
import { Check } from "lucide-react-native";
import { FlatList, Pressable, View } from "react-native";
import { useCSSVariable } from "uniwind";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { SheetTitle } from "@/components/SheetTitle";
import { Text } from "@/components/ui/text";
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
  const [foreground] = useCSSVariable(["--color-foreground"]);
  const { ticketId, projectId, currentStatusId } = useLocalSearchParams<StatusPickerParams>();
  const { data: statuses, error: statusesError, isPending } = useFetchProjectStatuses(projectId);
  const { data: ticket } = useFetchTicket(ticketId);
  const { data: project } = useFetchProject(projectId);
  const updateStatus = useUpdateTicketStatus();

  return (
    <View className="bg-popover pb-6">
      <SheetTitle title={ticket ? `Status · ${ticketKey(ticket, project?.prefix)}` : "Status"} />
      {isPending && <LoadingDisplay />}
      {statusesError && <ErrorDisplay error={statusesError} />}
      {updateStatus.error && <ErrorDisplay error={updateStatus.error} />}
      {statuses && (
        <FlatList
          data={statuses}
          keyExtractor={(s) => s.id}
          renderItem={({ item }) => (
            <Pressable
              role="button"
              disabled={updateStatus.isPending}
              onPress={() => updateStatus.mutate({ id: ticketId, status: item.id }, { onSuccess: () => router.back() })}
              className="min-h-11 flex-row items-center gap-2 border-b border-border px-4 active:bg-accent disabled:opacity-50"
            >
              <View className={cn("size-2 rounded-full", statusStageDot(item.kind))} />
              <Text className="min-w-0 flex-1 font-medium" numberOfLines={1}>
                {item.name}
              </Text>
              {item.id === currentStatusId && <Check size={16} color={String(foreground)} />}
            </Pressable>
          )}
        />
      )}
    </View>
  );
};
