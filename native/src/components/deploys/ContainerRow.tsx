import { ChevronRight } from "lucide-react-native";
import { Pressable, View } from "react-native";
import { useCSSVariable } from "uniwind";

import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";
import { containerImage, containerStatusDot, type Container } from "@/models/Stack";

interface ContainerRowProps {
  container: Container;
  // Set only for a viewer who can read logs; without it the row is plain and nothing hints at a destination.
  onOpenLogs?: (() => void) | undefined;
}

// Read-only: the compose file is the only source of truth for a container (spec §2), no editing on the phone.
export const ContainerRow = ({ container, onOpenLogs }: ContainerRowProps) => {
  const [mutedForeground] = useCSSVariable(["--color-muted-foreground"]);
  return (
    <Pressable
      role={onOpenLogs ? "button" : undefined}
      accessibilityLabel={onOpenLogs ? `Logs for ${container.name}` : undefined}
      disabled={!onOpenLogs}
      onPress={onOpenLogs}
      className={cn("min-h-11 flex-row items-center gap-2.5 border-b border-border py-2.5", onOpenLogs && "active:bg-accent")}
    >
      <View className={cn("size-2 shrink-0 rounded-full", containerStatusDot(container.status))} />
      <View className="min-w-0 flex-1">
        <Text className="font-medium" numberOfLines={1}>
          {container.name}
        </Text>
        <Text variant="muted" className="font-mono text-xs" numberOfLines={1}>
          {containerImage(container)}
        </Text>
      </View>
      <Text variant="small" className="shrink-0 text-muted-foreground">
        {container.status}
      </Text>
      {onOpenLogs && <ChevronRight size={16} color={String(mutedForeground)} />}
    </Pressable>
  );
};
