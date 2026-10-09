import Server from "lucide-react-native/icons/server";
import { View } from "react-native";

import { StatusTile } from "@/components/StatusTile";
import { Text } from "@/components/ui/text";
import { runnerStatusDot, type Runner } from "@/models/Runner";

interface RunnerRowProps {
  runner: Runner;
}

const doing = (runner: Runner) => {
  if (!runner.connected) return "offline";
  if (!runner.running_job) return "idle";
  return runner.running_job.service ? `${runner.running_job.kind} · ${runner.running_job.service}` : runner.running_job.kind;
};

// An offline runner keeps full-contrast text and loses its tile's fill for a dashed edge, so it reads unplugged, not faded.
export const RunnerRow = ({ runner }: RunnerRowProps) => (
  <View accessible className="min-h-16 flex-row items-center gap-3.5 px-5 py-3">
    <StatusTile icon={Server} dot={runnerStatusDot(runner.connected)} dashed={!runner.connected} />
    <View className="min-w-0 flex-1 gap-0.5">
      <Text className="text-[15px] font-medium" numberOfLines={1}>
        {runner.name || runner.id}
      </Text>
      {runner.machine && runner.machine !== runner.name && (
        <Text className="font-mono text-xs text-muted-foreground" numberOfLines={1}>
          {runner.machine}
        </Text>
      )}
    </View>
    <View className="items-end gap-0.5">
      <Text className="text-[13px]">{doing(runner)}</Text>
      <Text className="font-mono text-xs text-muted-foreground">{runner.version}</Text>
    </View>
  </View>
);
