import { useRouter } from "expo-router";
import ChevronDown from "lucide-react-native/icons/chevron-down";
import { View } from "react-native";
import { useCSSVariable } from "uniwind";

import { ActionButton } from "@/components/action/ActionButton";
import { Text } from "@/components/ui/text";
import { useActionState } from "@/hooks/useActionState";
import { cn } from "@/lib/utils";
import { statusStageDot, type BoardStatus } from "@/models/Status";
import type { Ticket } from "@/models/Ticket";

interface TicketStatusFieldProps {
  ticket: Ticket;
  status: BoardStatus | undefined;
}

// The ticket's status as the button that opens the picker; a role that can't change it still reads the status.
export const TicketStatusField = ({ ticket, status }: TicketStatusFieldProps) => {
  const router = useRouter();
  const [muted] = useCSSVariable(["--color-muted-foreground"]);
  const action = useActionState({ permission: "tickets:write", projectId: ticket.project_id });
  const label = (
    <>
      {status && <View className={cn("size-2 rounded-full", statusStageDot(status.kind))} />}
      <Text className="text-sm">{status?.name ?? "Status"}</Text>
    </>
  );
  return (
    <>
      {action.state === "hidden" && <View className="flex-row items-center gap-2">{label}</View>}
      {action.state !== "hidden" && (
        <ActionButton
          permission="tickets:write"
          projectId={ticket.project_id}
          verb="change a ticket's status"
          name="Change status"
          variant="outline"
          size="sm"
          className="-my-1 min-h-11"
          aria-label={`Status, ${status?.name ?? "none"}${action.state === "live" ? ". Change" : ""}`}
          onPress={() =>
            router.push({
              pathname: "/board/status-picker",
              params: { ticketId: ticket.id, projectId: ticket.project_id, currentStatusId: ticket.status },
            })
          }
        >
          {label}
          {action.state === "live" && <ChevronDown size={14} color={String(muted)} />}
        </ActionButton>
      )}
    </>
  );
};
