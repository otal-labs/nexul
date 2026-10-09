import { Pressable, View } from "react-native";

import { RelativeTime } from "@/components/RelativeTime";
import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";
import { deployStatusDot, deployTitle, type Deploy } from "@/models/Stack";

interface DeployRowProps {
  deploy: Deploy;
  first: boolean;
  onPress: () => void;
}

export const DeployRow = ({ deploy, first, onPress }: DeployRowProps) => (
  <Pressable
    role="button"
    onPress={onPress}
    className={cn("min-h-12 flex-row items-center gap-3 px-4 py-2.5 active:bg-accent", !first && "border-t border-border")}
  >
    <View className={cn("size-2 rounded-full", deployStatusDot(deploy.status))} />
    <Text className="w-16 text-[13px]">{deploy.status}</Text>
    <Text className="min-w-0 flex-1 font-mono text-xs text-muted-foreground" numberOfLines={1}>
      {deployTitle(deploy)}
    </Text>
    <Text className="font-mono text-xs text-muted-foreground">
      <RelativeTime iso={deploy.created_at} />
    </Text>
  </Pressable>
);
