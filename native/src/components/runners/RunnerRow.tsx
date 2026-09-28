import { View } from "react-native";

import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";
import { runnerStatusDot, type Runner } from "@/models/Runner";

interface RunnerRowProps {
  runner: Runner;
}

export const RunnerRow = ({ runner }: RunnerRowProps) => (
  <View className="min-h-11 flex-row items-center gap-2.5 border-b border-border px-4 py-3">
    <View className={cn("size-2 shrink-0 rounded-full", runnerStatusDot(runner.connected))} />
    <View className="min-w-0 flex-1">
      <Text className="font-medium" numberOfLines={1}>
        {runner.name || runner.id}
      </Text>
      {runner.machine && (
        <Text variant="muted" className="font-mono text-xs" numberOfLines={1}>
          {runner.machine}
        </Text>
      )}
    </View>
    <Text variant="small" className="shrink-0 font-mono text-muted-foreground">
      {runner.version}
    </Text>
  </View>
);
