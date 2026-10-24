import { useRouter } from "expo-router";
import Ticket from "lucide-react-native/icons/ticket";
import { Pressable, View } from "react-native";

import { StatusTile } from "@/components/StatusTile";
import { Text } from "@/components/ui/text";
import { useFetchProject } from "@/hooks/ProjectHooks";
import { ticketKey, type Ticket as TicketModel } from "@/models/Ticket";

// A ticket due that day, which lasts all day, so it carries "Due" where a meeting carries its hours.
export const AgendaDueRow = ({ ticket }: { ticket: TicketModel }) => {
  const router = useRouter();
  const { data: project } = useFetchProject(ticket.project_id);
  return (
    <Pressable
      role="button"
      onPress={() => router.push(`/board/ticket/${ticket.id}`, { withAnchor: true })}
      className="min-h-16 flex-row items-center gap-3.5 px-5 py-2.5 active:bg-accent"
    >
      <StatusTile icon={Ticket} />
      <View className="min-w-0 flex-1 gap-0.5">
        <Text numberOfLines={2} className="text-[15px] leading-5">
          {ticket.title}
        </Text>
        {project && <Text className="font-mono text-xs text-muted-foreground">{ticketKey(ticket, project.prefix)}</Text>}
      </View>
      <Text className="font-mono text-xs text-muted-foreground">Due</Text>
    </Pressable>
  );
};
