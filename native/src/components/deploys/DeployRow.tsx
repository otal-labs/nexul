import { Pressable, View } from "react-native";

import { Text } from "@/components/ui/text";
import { formatRelativeTime } from "@/lib/time";
import { cn } from "@/lib/utils";
import { deployStatusDot, type Deploy } from "@/models/Stack";

interface DeployRowProps {
  deploy: Deploy;
  onPress: () => void;
}

export const DeployRow = ({ deploy, onPress }: DeployRowProps) => (
  <Pressable
    role="button"
    onPress={onPress}
    className="min-h-11 flex-row items-center gap-2.5 border-b border-border py-2.5 active:bg-accent"
  >
    <View className={cn("size-2 shrink-0 rounded-full", deployStatusDot(deploy.status))} />
    <Text className="min-w-0 flex-1 font-mono text-xs" numberOfLines={1}>
      {deploy.image || "repo build"}
    </Text>
    <Text variant="small" className="shrink-0 font-mono text-muted-foreground">
      {formatRelativeTime(deploy.created_at)}
    </Text>
  </Pressable>
);
