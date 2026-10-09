import Layers from "lucide-react-native/icons/layers";
import { Pressable, View } from "react-native";

import { RelativeTime } from "@/components/RelativeTime";
import { StatusTile } from "@/components/StatusTile";
import { Text } from "@/components/ui/text";
import { deployStatusDot, type Deploy, type Stack } from "@/models/Stack";

interface StackRowProps {
  stack: Stack;
  latest: Deploy | undefined;
  onPress: () => void;
}

// The stack's last deploy as the dot on its tile, the name over its machine, the deploy's state and age trailing.
export const StackRow = ({ stack, latest, onPress }: StackRowProps) => (
  <Pressable role="button" onPress={onPress} className="min-h-16 flex-row items-center gap-3.5 px-5 py-3 active:bg-accent">
    <StatusTile icon={Layers} dot={latest && deployStatusDot(latest.status)} />
    <View className="min-w-0 flex-1 gap-0.5">
      <Text className="text-[15px] font-medium" numberOfLines={1}>
        {stack.name}
      </Text>
      <Text className="font-mono text-xs text-muted-foreground" numberOfLines={1}>
        {stack.machine}
      </Text>
    </View>
    <View className="items-end gap-0.5">
      {latest && <Text className="text-[13px]">{latest.status}</Text>}
      {latest && (
        <Text className="font-mono text-xs text-muted-foreground">
          <RelativeTime iso={latest.created_at} />
        </Text>
      )}
      {!latest && <Text className="text-[13px] text-muted-foreground">No deploys</Text>}
    </View>
  </Pressable>
);
