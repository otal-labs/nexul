import { clarificationStatus, type Clarification, type ClarificationStatus } from "@nexul/client-core/clarification";
import CircleCheck from "lucide-react-native/icons/circle-check";
import { View } from "react-native";

import { Icon } from "@/components/ui/icon";
import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";

interface ClarificationStatusLineProps {
  clarification: Clarification;
  locked: boolean;
}

const StatusIcon = ({ icon }: { icon: ClarificationStatus["icon"] }) => (
  <>
    {icon === "waiting" && <View className="size-2 rounded-full bg-info" />}
    {icon === "running" && <View className="size-2 rounded-full bg-warning" />}
    {icon === "done" && <Icon as={CircleCheck} size={16} className="text-success" />}
  </>
);

// The clarification's state as one line under the heading; whoever may close it alone sees a running round and "No gaps left".
export const ClarificationStatusLine = ({ clarification, locked }: ClarificationStatusLineProps) => {
  const { icon, label, detail } = clarificationStatus(clarification, clarification.can_close, locked);
  return (
    <View className="flex-row flex-wrap items-center gap-x-2 gap-y-0.5">
      <StatusIcon icon={icon} />
      <Text className={cn("text-sm leading-snug", icon ? "font-medium" : "text-muted-foreground")}>{label}</Text>
      {detail !== "" && <Text className="text-sm leading-snug text-muted-foreground">· {detail}</Text>}
    </View>
  );
};
