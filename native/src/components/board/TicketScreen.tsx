import { Stack, useLocalSearchParams, useRouter } from "expo-router";
import ChevronDown from "lucide-react-native/icons/chevron-down";
import MessageSquare from "lucide-react-native/icons/message-square";
import { Pressable, ScrollView, View } from "react-native";
import { useCSSVariable } from "uniwind";

import { TicketPerson } from "@/components/board/TicketPeople";
import { ticketTypePill } from "@/components/board/ticketTypeColor";
import { DocBody } from "@/components/docs/DocBody";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { FactRow } from "@/components/FactRow";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Microheader } from "@/components/Microheader";
import { ProjectRevokedGate } from "@/components/project/ProjectRevokedGate";
import { ScreenHeader } from "@/components/ScreenHeader";
import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useFetchProject } from "@/hooks/ProjectHooks";
import { useFetchProjectStatuses } from "@/hooks/StatusHooks";
import { useFetchTicket, useOpenTicketThread, useSetTicketPerson } from "@/hooks/TicketHooks";
import { useFetchProjectTicketTypes } from "@/hooks/TicketTypeHooks";
import { useCurrentWorkspaceId } from "@/hooks/WorkspaceHooks";
import { cn } from "@/lib/utils";
import { statusStageDot } from "@/models/Status";
import { ticketKey, TicketRole } from "@/models/Ticket";

// The record's mono id under its title, the web detail header's id chip.
const TicketKey = ({ label }: { label: string }) => <Text className="font-mono text-xs text-muted-foreground">{label}</Text>;

export const TicketScreen = () => {
  const router = useRouter();
  const { id, workspace } = useLocalSearchParams<{ id: string; workspace?: string }>();
  const { data: ticket, error, isPending } = useFetchTicket(id, workspace);
  const { data: project } = useFetchProject(ticket?.project_id);
  const { data: statuses } = useFetchProjectStatuses(ticket?.project_id);
  const { data: ticketTypes } = useFetchProjectTicketTypes(ticket?.project_id);
  const { data: me } = useFetchMe(true);
  const workspaceId = useCurrentWorkspaceId();
  const setPerson = useSetTicketPerson();
  const openThread = useOpenTicketThread();
  const [muted] = useCSSVariable(["--color-muted-foreground"]);

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
    <ProjectRevokedGate projectId={ticket?.project_id}>
      <ScrollView className="flex-1 bg-background" contentContainerClassName="pb-8">
        <Stack.Screen options={{ title: ticket && project ? ticketKey(ticket, project.prefix) : "" }} />
        {isPending && <LoadingDisplay message="Loading the ticket" />}
        {error && <ErrorDisplay error={error} notFound="This ticket doesn't exist or was deleted." />}
        {ticket && <ScreenHeader eyebrow={project?.name} title={ticket.title} meta={<TicketKey label={ticketKey(ticket, project?.prefix)} />} className="pt-2" />}
        {ticket && (
          <View className="gap-5 px-4">
            <View className="overflow-hidden rounded-xl border border-border bg-card">
              <FactRow label="Status" first>
                <Pressable
                  role="button"
                  aria-label={`Status, ${status?.name ?? "none"}. Change`}
                  onPress={() =>
                    router.push({
                      pathname: "/board/status-picker",
                      params: { ticketId: ticket.id, projectId: ticket.project_id, currentStatusId: ticket.status },
                    })
                  }
                  className="-my-1 min-h-11 flex-row items-center gap-2 rounded-md border border-input px-3 active:bg-accent"
                >
                  {status && <View className={cn("size-2 rounded-full", statusStageDot(status.kind))} />}
                  <Text className="text-sm">{status?.name ?? "Status"}</Text>
                  <ChevronDown size={14} color={String(muted)} />
                </Pressable>
              </FactRow>
              <FactRow label="Type">
                {type && <Text className={cn("overflow-hidden rounded-full px-2 py-0.5 text-xs font-medium", ticketTypePill(type.name, type.color))}>{type.name}</Text>}
                {!type && <Text className="text-sm text-muted-foreground">None</Text>}
              </FactRow>
              <FactRow label="Developer">
                <TicketPerson login={ticket.developer} />
              </FactRow>
              <FactRow label="Tester">
                <TicketPerson login={ticket.tester} />
              </FactRow>
            </View>

            {setPerson.error && <ErrorDisplay error={setPerson.error} className="px-0" />}
            {openThread.error && <ErrorDisplay error={openThread.error} className="px-0" />}
            <View className="flex-row gap-2">
              {ticket.developer !== me?.user.login && (
                <Button variant="outline" className="flex-1" onPress={assignToMe} disabled={setPerson.isPending}>
                  <Text>Assign to me</Text>
                </Button>
              )}
              <Button variant="outline" className="flex-1" onPress={openInChat} disabled={openThread.isPending || !workspaceId}>
                <MessageSquare size={16} color={String(muted)} />
                <Text>Open thread</Text>
              </Button>
            </View>

            <View className="gap-2">
              <Microheader>Description</Microheader>
              <DocBody body={ticket.body} />
            </View>
          </View>
        )}
      </ScrollView>
    </ProjectRevokedGate>
  );
};
