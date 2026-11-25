import { Stack, useLocalSearchParams } from "expo-router";
import { ScrollView, View } from "react-native";

import { isTrailActive, trailSummary } from "@nexul/client-core/trail";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { RunContinueForm } from "@/components/plays/RunContinueForm";
import { RunStateDot } from "@/components/plays/RunStateDot";
import { RunTranscript } from "@/components/plays/RunTranscript";
import { RelativeTime } from "@/components/RelativeTime";
import { ScreenHeader } from "@/components/ScreenHeader";
import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import { useCanActOnRun, useFetchTrail, useStopRun } from "@/hooks/TrailHooks";
import type { RunTab } from "@/models/Play";
import type { Trail } from "@/models/Trail";

// How the run stands, in the web's word and hue, and when it started.
const RunStatus = ({ trail }: { trail: Trail }) => (
  <View className="flex-row items-center gap-2">
    <RunStateDot state={trail.state} />
    <Text numberOfLines={1} className="shrink text-row text-muted-foreground">
      {trailSummary(trail.state, "", null)}
    </Text>
    <Text className="font-mono text-xs text-muted-foreground">
      <RelativeTime iso={trail.started_at} />
    </Text>
  </View>
);

// The run's steps and question first, then Stop while it works or waits, or Continue once it ended; both for its starter or a plays:write holder only.
const RunBody = ({ trail, tab }: { trail: Trail; tab: RunTab }) => {
  const canAct = useCanActOnRun(trail);
  const stop = useStopRun();
  const active = isTrailActive(trail.state);
  return (
    <View className="gap-5 px-4">
      {trail.last_error !== "" && <Text className="text-row text-destructive">{trail.last_error}</Text>}
      <RunTranscript trail={trail} />
      {active && canAct && (
        <Button variant="outline" size="sm" className="self-start" disabled={stop.isPending} onPress={() => stop.mutate(trail.id)}>
          <Text>{stop.isPending ? "Stopping…" : "Stop run"}</Text>
        </Button>
      )}
      {stop.error && <ErrorDisplay error={stop.error} className="px-0" />}
      {!active && canAct && <RunContinueForm trail={trail} tab={tab} />}
    </View>
  );
};

// One run of a play, followed live: its frames patch the cached run, so the steps, the question and the end land here.
export const RunScreen = ({ tab }: { tab: RunTab }) => {
  const { id } = useLocalSearchParams<{ id: string }>();
  const { data: trail, error, isPending } = useFetchTrail(id);
  return (
    <ScrollView className="flex-1 bg-background" contentContainerClassName="pb-8" keyboardShouldPersistTaps="handled">
      <Stack.Screen options={{ title: "Run" }} />
      {isPending && <LoadingDisplay message="Loading the run" />}
      {error && <ErrorDisplay error={error} notFound="This run doesn't exist, or you can't see it." />}
      {trail && <ScreenHeader title={trail.play_label} meta={<RunStatus trail={trail} />} className="pt-2" />}
      {trail && <RunBody trail={trail} tab={tab} />}
    </ScrollView>
  );
};
