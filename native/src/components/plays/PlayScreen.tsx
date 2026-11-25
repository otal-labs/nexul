import { Stack, useLocalSearchParams } from "expo-router";
import { ScrollView, View } from "react-native";

import { PLAY_STAGE_LABELS } from "@nexul/client-core/plays";

import { AreaGate } from "@/components/AreaGate";
import { EmptyState } from "@/components/EmptyState";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { FactRow } from "@/components/FactRow";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { AutoPlaysSection } from "@/components/plays/AutoPlaysSection";
import { PlayEnabledSwitch } from "@/components/plays/PlayEnabledSwitch";
import { PlayHealthLine } from "@/components/plays/PlayHealthLine";
import { PlayQueuedSection } from "@/components/plays/PlayQueuedSection";
import { PlayRunsSection } from "@/components/plays/PlayRunsSection";
import { ScreenHeader } from "@/components/ScreenHeader";
import { Text } from "@/components/ui/text";
import { useFetchWorkspacePlays } from "@/hooks/PlayHooks";
import type { WorkspacePlay } from "@/models/Play";

const TYPE_LABELS = { ticket: "Ticket play", doc: "Doc play", interview: "Interview play", stack: "Stack play" } as const;

// "Ticket play · on In progress": what it runs on, and where a ticket play shows.
const playMeta = (play: WorkspacePlay) =>
  [TYPE_LABELS[play.type], play.type === "ticket" && play.show_when_stage && `on ${PLAY_STAGE_LABELS[play.show_when_stage]}`].filter(Boolean).join(" · ");

// One play: its switch and health, its runs, what it has queued, and its auto plays, read from the plays list the screen came from.
export const PlayScreen = () => {
  const { id } = useLocalSearchParams<{ id: string }>();
  const { data, error, isPending } = useFetchWorkspacePlays();
  const play = data?.find((p) => p.id === id);
  return (
    <AreaGate area="plays">
      <ScrollView className="flex-1 bg-background" contentContainerClassName="pb-8">
        <Stack.Screen options={{ title: "Play" }} />
        {isPending && <LoadingDisplay message="Loading the play" />}
        {error && <ErrorDisplay error={error} />}
        {data && !play && <EmptyState size="compact" title="This play is gone" message="It was deleted, or your role no longer sees it." />}
        {play && <ScreenHeader title={play.label} meta={playMeta(play)} className="pt-2" />}
        {play && (
          <View className="gap-5 px-4">
            {play.description !== "" && <Text className="text-body text-muted-foreground">{play.description}</Text>}
            <View className="overflow-hidden rounded-xl border border-border bg-card">
              <FactRow label="Enabled" first>
                <PlayEnabledSwitch play={play} />
              </FactRow>
              <FactRow label="Runs">
                <View className="min-w-0 flex-1 items-end">
                  <PlayHealthLine health={play} />
                </View>
              </FactRow>
            </View>
            <PlayRunsSection playId={play.id} />
            <PlayQueuedSection playId={play.id} />
            {play.type !== "interview" && <AutoPlaysSection play={play} />}
          </View>
        )}
      </ScrollView>
    </AreaGate>
  );
};
