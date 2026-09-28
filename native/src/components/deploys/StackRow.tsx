import { Pressable, View } from "react-native";

import { Text } from "@/components/ui/text";
import { formatRelativeTime } from "@/lib/time";
import { cn } from "@/lib/utils";
import { deployStatusDot, type Deploy, type Stack } from "@/models/Stack";

interface StackRowProps {
  stack: Stack;
  latest: Deploy | undefined;
  onPress: () => void;
}

// Hairline row per practices/native.md: primary field (name) left, mono meta (target, last deploy) right.
export const StackRow = ({ stack, latest, onPress }: StackRowProps) => (
  <Pressable
    role="button"
    onPress={onPress}
    className="min-h-11 flex-row items-center gap-2.5 border-b border-border px-4 py-3 active:bg-accent"
  >
    <View className={cn("size-2 shrink-0 rounded-full", deployStatusDot(latest?.status))} />
    <View className="min-w-0 flex-1">
      <Text className="font-medium" numberOfLines={1}>
        {stack.name}
      </Text>
      <Text variant="muted" className="font-mono text-xs" numberOfLines={1}>
        {stack.machine}
      </Text>
    </View>
    {latest && (
      <Text variant="small" className="shrink-0 font-mono text-muted-foreground">
        {formatRelativeTime(latest.created_at)}
      </Text>
    )}
    {!latest && <Text variant="small" className="shrink-0 text-muted-foreground">No deploys</Text>}
  </Pressable>
);
