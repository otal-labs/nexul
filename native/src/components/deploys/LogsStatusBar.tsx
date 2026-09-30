import { View } from "react-native";

import { Text } from "@/components/ui/text";
import type { LogStatus } from "@/hooks/ContainerLogHooks";
import { cn } from "@/lib/utils";

interface LogsStatusBarProps {
  status: LogStatus;
  following: boolean;
}

const describe = (status: LogStatus, following: boolean): { label: string; dot: string } => {
  if (status === "offline") return { label: "Runner offline, reconnecting…", dot: "bg-warning" };
  if (status === "ended") return { label: "Stream ended", dot: "bg-muted-foreground" };
  if (status === "connecting") return { label: "Connecting…", dot: "bg-muted-foreground" };
  if (!following) return { label: "Paused", dot: "bg-muted-foreground" };
  return { label: "Live", dot: "bg-success" };
};

export const LogsStatusBar = ({ status, following }: LogsStatusBarProps) => {
  const { label, dot } = describe(status, following);
  return (
    <View className="flex-row items-center gap-2 border-b border-border px-4 py-2">
      <View className={cn("size-2 rounded-full", dot)} />
      <Text variant="small" className="text-muted-foreground">
        {label}
      </Text>
    </View>
  );
};
