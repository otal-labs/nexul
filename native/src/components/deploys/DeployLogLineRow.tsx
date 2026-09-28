import { View } from "react-native";

import { Text } from "@/components/ui/text";
import { formatLogTimestamp } from "@/lib/time";
import type { DeployLogLine } from "@/models/Stack";

interface DeployLogLineRowProps {
  line: DeployLogLine;
}

export const DeployLogLineRow = ({ line }: DeployLogLineRowProps) => (
  <View className="flex-row gap-3 px-3 py-0.5">
    <Text className="shrink-0 font-mono text-xs text-muted-foreground">{formatLogTimestamp(line.ts)}</Text>
    <Text className="flex-1 font-mono text-xs">{line.text}</Text>
  </View>
);
