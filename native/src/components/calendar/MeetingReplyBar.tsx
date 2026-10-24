import Check from "lucide-react-native/icons/check";
import X from "lucide-react-native/icons/x";
import { View } from "react-native";
import { useCSSVariable } from "uniwind";

import { myReply, type Meeting, type Reply } from "@nexul/client-core/calendar";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useReplyToMeeting } from "@/hooks/CalendarHooks";
import { cn } from "@/lib/utils";

type Answer = Exclude<Reply, "pending">;

const STATUS: Record<Reply, { text: string; dot: string }> = {
  pending: { text: "You haven't replied", dot: "bg-muted-foreground" },
  accepted: { text: "You accepted", dot: "bg-success" },
  declined: { text: "You declined", dot: "bg-destructive" },
};

interface ReplyButtonProps {
  label: string;
  icon: typeof Check;
  chosen: boolean;
  busy: boolean;
  onPress: () => void;
}

// The answer already given is the filled button, and pressing it again does nothing.
const ReplyButton = ({ label, icon: Icon, chosen, busy, onPress }: ReplyButtonProps) => {
  const [foreground, brandForeground] = useCSSVariable(["--color-foreground", "--color-brand-foreground"]);
  return (
    <Button
      variant={chosen ? "default" : "outline"}
      className="flex-1"
      accessibilityState={{ selected: chosen }}
      disabled={busy}
      onPress={() => !chosen && onPress()}
    >
      <Icon size={16} color={String(chosen ? brandForeground : foreground)} />
      <Text>{label}</Text>
    </Button>
  );
};

// Only an invitee replies; the reply covers every occurrence of a repeating meeting.
export const MeetingReplyBar = ({ meeting }: { meeting: Meeting }) => {
  const { data: me } = useFetchMe(true);
  const { mutate, isPending, error } = useReplyToMeeting();
  const current = me && myReply(meeting, me.user.id);
  if (!current) return null;
  const status = STATUS[current];
  const answer = (reply: Answer) => () => mutate({ id: meeting.id, reply });
  return (
    <View className="gap-3">
      <View className="flex-row items-center gap-2 px-1">
        <View className={cn("size-2 rounded-full", status.dot)} />
        <Text className="text-sm">{status.text}</Text>
      </View>
      <View className="flex-row gap-2">
        <ReplyButton label="Accept" icon={Check} chosen={current === "accepted"} busy={isPending} onPress={answer("accepted")} />
        <ReplyButton label="Decline" icon={X} chosen={current === "declined"} busy={isPending} onPress={answer("declined")} />
      </View>
      {error && <ErrorDisplay error={error} className="px-0" />}
    </View>
  );
};
