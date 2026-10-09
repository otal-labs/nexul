import ChevronRight from "lucide-react-native/icons/chevron-right";
import { Pressable, View } from "react-native";
import { useCSSVariable } from "uniwind";

import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";
import { containerImage, containerStatusDot, type Container } from "@/models/Stack";

interface ContainerRowProps {
  container: Container;
  first: boolean;
  // Set only for a viewer who can read logs; without it the row is plain and nothing hints at a destination.
  onOpenLogs?: (() => void) | undefined;
}

// Read-only: the compose file is the only source of truth for a container (spec §2), no editing on the phone.
export const ContainerRow = ({ container, first, onOpenLogs }: ContainerRowProps) => {
  const [mutedForeground] = useCSSVariable(["--color-muted-foreground"]);
  return (
    <Pressable
      role={onOpenLogs ? "button" : undefined}
      accessibilityLabel={onOpenLogs ? `Logs for ${container.name}` : undefined}
      disabled={!onOpenLogs}
      onPress={onOpenLogs}
      className={cn("min-h-14 flex-row items-center gap-3 px-4 py-2.5", !first && "border-t border-border", onOpenLogs && "active:bg-accent")}
    >
      <View className="min-w-0 flex-1 gap-0.5">
        <Text className="font-mono text-sm" numberOfLines={1}>
          {container.name}
        </Text>
        <Text className="font-mono text-xs text-muted-foreground" numberOfLines={1}>
          {containerImage(container)}
        </Text>
      </View>
      <View className={cn("size-2 rounded-full", containerStatusDot(container.status))} />
      <Text className="text-[13px] text-muted-foreground">{container.status}</Text>
      {onOpenLogs && <ChevronRight size={16} color={String(mutedForeground)} />}
    </Pressable>
  );
};
