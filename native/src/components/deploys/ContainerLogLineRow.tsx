import { View } from "react-native";

import { Text } from "@/components/ui/text";
import { formatLogTimestamp } from "@/lib/time";
import { cn } from "@/lib/utils";
import type { ContainerLogLine } from "@/models/Stack";

interface ContainerLogLineRowProps {
  line: ContainerLogLine;
}

// stderr is the only color in the view: a thin status gutter, never a tint on the text.
export const ContainerLogLineRow = ({ line }: ContainerLogLineRowProps) => (
  <View className="flex-row gap-2.5 pr-3">
    <View className={cn("w-0.5 self-stretch", line.stream === "stderr" && "bg-destructive")} />
    <Text className="shrink-0 py-0.5 font-mono text-xs text-muted-foreground">{formatLogTimestamp(Date.parse(line.ts))}</Text>
    <Text className="flex-1 py-0.5 font-mono text-xs">{line.line}</Text>
  </View>
);
