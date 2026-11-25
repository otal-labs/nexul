import { useRouter } from "expo-router";
import { memo } from "react";
import { Pressable, View } from "react-native";

import { trailSummary } from "@nexul/client-core/trail";

import { RunStateDot } from "@/components/plays/RunStateDot";
import { RelativeTime } from "@/components/RelativeTime";
import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";
import { runRoute, type RunTab } from "@/models/Play";
import type { Trail } from "@/models/Trail";

interface RunRowProps {
  run: Trail;
  // What the row names first: the play on a ticket's list, the ticket or doc on a list across targets.
  title: string;
  tab: RunTab;
  first: boolean;
}

// The state dot, what it ran, its newest step or how it ended, and when it started; a frame for another run leaves it alone.
export const RunRow = memo(({ run, title, tab, first }: RunRowProps) => {
  const router = useRouter();
  return (
    <Pressable
      role="button"
      aria-label={`${title}, ${trailSummary(run.state, run.last_error, run.activity.at(-1))}`}
      onPress={() => router.push(runRoute(tab, run.id))}
      className={cn("min-h-[52px] flex-row items-center gap-3 px-4 py-2.5 active:bg-accent", !first && "border-t border-border")}
    >
      <RunStateDot state={run.state} />
      <View className="min-w-0 flex-1 gap-0.5">
        <Text numberOfLines={1} className="text-body">
          {title}
        </Text>
        <Text numberOfLines={1} className="text-row text-muted-foreground">
          {trailSummary(run.state, run.last_error, run.activity.at(-1))}
        </Text>
      </View>
      <Text className="font-mono text-xs text-muted-foreground">
        <RelativeTime iso={run.started_at} />
      </Text>
    </Pressable>
  );
});
RunRow.displayName = "RunRow";
