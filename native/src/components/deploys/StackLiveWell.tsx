import { View } from "react-native";

import { Microheader } from "@/components/Microheader";
import { RelativeTime } from "@/components/RelativeTime";
import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";
import { ContainerStatus, deployStatusDot, type Container, type Deploy } from "@/models/Stack";

interface StackLiveWellProps {
  latest: Deploy | undefined;
  containers: Container[] | undefined;
}

const UP: string[] = [ContainerStatus.Running, ContainerStatus.Healthy];

// The stack's live status as one well split in two: the last deploy's state, and how many services are up.
export const StackLiveWell = ({ latest, containers }: StackLiveWellProps) => {
  const up = containers?.filter((c) => UP.includes(c.status)).length ?? 0;
  return (
    <View className="flex-row rounded-xl border border-border bg-surface-2">
      <View className="flex-1 gap-1.5 p-4">
        <Microheader>Last deploy</Microheader>
        <View className="flex-row items-center gap-2">
          <View className={cn("size-2.5 rounded-full", deployStatusDot(latest?.status))} />
          <Text className="text-xl font-semibold capitalize">{latest?.status ?? "None"}</Text>
        </View>
        {latest && (
          <Text className="font-mono text-xs text-muted-foreground">
            <RelativeTime iso={latest.created_at} />
          </Text>
        )}
      </View>
      <View className="w-px bg-border" />
      <View className="flex-1 gap-1.5 p-4">
        <Microheader>Services</Microheader>
        <Text className="font-mono text-xl">{containers?.length ?? "–"}</Text>
        {containers && <Text className="font-mono text-xs text-muted-foreground">{up} up</Text>}
      </View>
    </View>
  );
};
