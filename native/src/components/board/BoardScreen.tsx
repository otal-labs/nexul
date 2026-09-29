import { useRouter } from "expo-router";
import { ChevronDown } from "lucide-react-native";
import { useState } from "react";
import { Pressable, ScrollView, View } from "react-native";
import { useCSSVariable } from "uniwind";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Text } from "@/components/ui/text";
import { StatusSection } from "@/components/board/StatusSection";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useFetchProjectStatuses } from "@/hooks/StatusHooks";
import { useFetchTicketsByProject } from "@/hooks/TicketHooks";
import { useFetchProjectTicketTypes } from "@/hooks/TicketTypeHooks";
import { cn } from "@/lib/utils";
import { effectiveProject } from "@/models/Project";
import { useBoardStore } from "@/stores/boardStore";

export const BoardScreen = () => {
  const router = useRouter();
  const [mutedForeground] = useCSSVariable(["--color-muted-foreground"]);
  const [mineOnly, setMineOnly] = useState(false);
  const { data: me } = useFetchMe(true);
  const { data: projects, error: projectsError, isPending: projectsPending } = useFetchProjects();
  const selectedProjectId = useBoardStore((s) => s.selectedProjectId);
  const project = effectiveProject(projects, selectedProjectId);
  const { data: statuses, error: statusesError, isPending: statusesPending } = useFetchProjectStatuses(project?.id);
  const { data: tickets, error: ticketsError, isPending: ticketsPending } = useFetchTicketsByProject(project?.id);
  const { data: ticketTypes } = useFetchProjectTicketTypes(project?.id);

  const isPending = projectsPending || (!!project && (statusesPending || ticketsPending));
  const error = projectsError ?? statusesError ?? ticketsError;
  const filteredTickets =
    mineOnly && me ? tickets?.filter((t) => t.developer === me.user.login || t.tester === me.user.login) : tickets;

  return (
    <View className="flex-1 bg-background">
      {projects && projects.length > 0 && (
        <View className="flex-row items-center gap-2 border-b border-border px-4 py-3">
          <Pressable
            role="button"
            onPress={() => router.push("/board/project-picker")}
            className="min-h-11 flex-1 flex-row items-center gap-1.5 active:opacity-70"
          >
            <Text className="min-w-0 flex-1 font-semibold" numberOfLines={1}>
              {project?.name ?? "Choose a project"}
            </Text>
            <ChevronDown size={16} color={String(mutedForeground)} />
          </Pressable>
          <Pressable
            role="button"
            onPress={() => setMineOnly((v) => !v)}
            aria-label="Mine"
            aria-pressed={mineOnly}
            className={cn(
              "min-h-11 min-w-11 items-center justify-center rounded-md border border-border px-3",
              mineOnly && "border-primary bg-primary",
            )}
          >
            <Text variant="small" className={cn(mineOnly && "font-semibold text-primary-foreground")}>
              Mine
            </Text>
          </Pressable>
        </View>
      )}
      {isPending && <LoadingDisplay message="Loading the board…" />}
      {error && <ErrorDisplay error={error} />}
      {projects && projects.length === 0 && (
        <View className="flex-1 items-center justify-center px-6">
          <Text variant="muted" className="text-center">
            No projects yet.
          </Text>
        </View>
      )}
      {project && statuses && filteredTickets && (
        <ScrollView className="flex-1">
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
        </ScrollView>
      )}
    </View>
  );
};
