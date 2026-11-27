import { useRouter } from "expo-router";
import { Pressable, View } from "react-native";

import { EmptyRow } from "@/components/EmptyRow";
import { Microheader } from "@/components/Microheader";
import { QueuedRunRow } from "@/components/plays/QueuedRunRow";
import { RunRow } from "@/components/plays/RunRow";
import { SettingsCard } from "@/components/SettingsCard";
import { useQueuedAcross } from "@/hooks/PlayQueueHooks";
import { useNowRuns } from "@/hooks/TrailHooks";
import type { PlayQueueItem, WorkspacePlay } from "@/models/Play";
import type { PlayRun } from "@/models/Trail";

// "WEB-12 · Fix with AI": where it runs, then what runs there.
const runTitle = (run: PlayRun) => [run.target_key || run.target_title, run.play_label].filter(Boolean).join(" · ");

// A queued ticket run opens its ticket on the Board, where the run will show once it starts.
const QueuedTicketRow = ({ item, first }: { item: PlayQueueItem; first: boolean }) => {
  const router = useRouter();
  const row = <QueuedRunRow item={item} title={item.play_label} first={first} />;
  return (
    <>
      {item.target_type === "ticket" && (
        <Pressable role="button" aria-label={`${item.play_label}, open its ticket`} onPress={() => router.push(`/board/ticket/${item.target_id}`, { withAnchor: true })} className="active:bg-accent">
          {row}
        </Pressable>
      )}
      {item.target_type !== "ticket" && row}
    </>
  );
};

// What runs across the workspace now, auto plays' runs included, then what waits in the queue to start.
export const NowSection = ({ plays }: { plays: WorkspacePlay[] }) => {
  const { runs, isPending } = useNowRuns(plays);
  const queued = useQueuedAcross(plays.filter((play) => (play.auto_plays_on ?? 0) > 0).map((play) => play.id));
  return (
    <View className="gap-5 px-4 pb-6">
      {!isPending && runs.length === 0 && (
        <View className="gap-2">
          <Microheader className="px-1">Now</Microheader>
          <EmptyRow message="Nothing running" />
        </View>
      )}
      {runs.length > 0 && (
        <SettingsCard title="Now">
          {runs.map((run, i) => (
            <RunRow key={run.id} run={run} title={runTitle(run)} tab="more" first={i === 0} />
          ))}
        </SettingsCard>
      )}
      {queued.length > 0 && (
        <SettingsCard title="Queued">
          {queued.map((item, i) => (
            <QueuedTicketRow key={item.id} item={item} first={i === 0} />
          ))}
        </SettingsCard>
      )}
    </View>
  );
};
