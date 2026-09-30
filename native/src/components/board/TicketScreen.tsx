import { Stack, useLocalSearchParams, useRouter } from "expo-router";
import { Pressable, ScrollView, View } from "react-native";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import { DocBody } from "@/components/docs/DocBody";
import { personLabel } from "@/models/Person";
import { statusStageDot } from "@/models/Status";
import { ticketKey, TicketRole } from "@/models/Ticket";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useFetchProject } from "@/hooks/ProjectHooks";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import { useCurrentWorkspaceId } from "@/hooks/WorkspaceHooks";
import { useFetchProjectStatuses } from "@/hooks/StatusHooks";
import { useFetchTicket, useOpenTicketThread, useSetTicketPerson } from "@/hooks/TicketHooks";
import { useFetchProjectTicketTypes } from "@/hooks/TicketTypeHooks";
import { cn } from "@/lib/utils";

export const TicketScreen = () => {
  const router = useRouter();
  const { id, workspace } = useLocalSearchParams<{ id: string; workspace?: string }>();
  const { data: ticket, error, isPending } = useFetchTicket(id, workspace);
  const { data: project } = useFetchProject(ticket?.project_id);
  const { data: statuses } = useFetchProjectStatuses(ticket?.project_id);
  const { data: ticketTypes } = useFetchProjectTicketTypes(ticket?.project_id);
  const { data: me } = useFetchMe(true);
  const workspaceId = useCurrentWorkspaceId();
  const resolvePerson = usePersonLookup(workspaceId);
  const setPerson = useSetTicketPerson();
  const openThread = useOpenTicketThread();

  const status = statuses?.find((s) => s.id === ticket?.status);
  const type = ticketTypes?.find((t) => t.id === ticket?.type_id);

  const assignToMe = () => {
    if (!ticket || !me) return;
    setPerson.mutate({ id: ticket.id, role: TicketRole.Developer, login: me.user.login });
  };

  const openInChat = () => {
    if (!ticket || !workspaceId) return;
    openThread.mutate(
      { id: ticket.id, workspaceId },
      { onSuccess: (conversation) => router.push(`/chat/${conversation.id}`, { withAnchor: true }) },
    );
  };

  return (
    <ScrollView className="flex-1 bg-background">
      <Stack.Screen options={{ title: ticket ? ticketKey(ticket, project?.prefix) : "Ticket" }} />
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} notFound="This ticket doesn't exist or was deleted." />}
      {ticket && (
        <View className="gap-4 px-4 py-4">
          <View className="flex-row items-center gap-2">
            <Text variant="small" className="font-mono text-muted-foreground">
              {ticketKey(ticket, project?.prefix)}
            </Text>
            <Pressable
              role="button"
              onPress={() =>
                router.push({
                  pathname: "/board/status-picker",
                  params: { ticketId: ticket.id, projectId: ticket.project_id, currentStatusId: ticket.status },
                })
              }
              className="min-h-11 flex-row items-center gap-1.5 rounded-md border border-border px-2.5 active:bg-accent"
            >
              {status && <View className={cn("size-2 rounded-full", statusStageDot(status.kind))} />}
              <Text variant="small">{status?.name ?? "Status"}</Text>
            </Pressable>
          </View>

          <Text variant="h3">{ticket.title}</Text>

          <View className="gap-1 border-t border-border pt-3">
            <View className="min-h-9 flex-row items-center gap-2">
              <Text variant="small" className="w-20 shrink-0 text-muted-foreground">
                Type
              </Text>
              <Text variant="small">{type?.name ?? "None"}</Text>
            </View>
            <View className="min-h-9 flex-row items-center gap-2">
              <Text variant="small" className="w-20 shrink-0 text-muted-foreground">
                Developer
              </Text>
              <Text variant="small">{ticket.developer ? personLabel(resolvePerson(ticket.developer)) : "No one"}</Text>
            </View>
            <View className="min-h-9 flex-row items-center gap-2">
              <Text variant="small" className="w-20 shrink-0 text-muted-foreground">
                Tester
              </Text>
              <Text variant="small">{ticket.tester ? personLabel(resolvePerson(ticket.tester)) : "No one"}</Text>
            </View>
          </View>

          {setPerson.error && <ErrorDisplay error={setPerson.error} className="px-0" />}
          {ticket.developer !== me?.user.login && (
            <Button variant="outline" onPress={assignToMe} disabled={setPerson.isPending}>
              <Text>Assign to me</Text>
            </Button>
          )}

          <View className="gap-2 border-t border-border pt-3">
            <Text variant="small" className="text-muted-foreground">
              Description
            </Text>
            <DocBody body={ticket.body} />
          </View>

          {openThread.error && <ErrorDisplay error={openThread.error} className="px-0" />}
          <Button variant="outline" onPress={openInChat} disabled={openThread.isPending || !workspaceId}>
            <Text>Open thread</Text>
          </Button>
        </View>
      )}
    </ScrollView>
  );
};
