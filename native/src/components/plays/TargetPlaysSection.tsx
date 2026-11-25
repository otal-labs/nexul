import { View } from "react-native";

import type { PlayStage } from "@nexul/client-core/plays";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { AutoPlaysPausedRow } from "@/components/plays/AutoPlaysPausedRow";
import { PlayStartRow } from "@/components/plays/PlayStartRow";
import { QueuedRunRow } from "@/components/plays/QueuedRunRow";
import { RunRow } from "@/components/plays/RunRow";
import { SettingsCard } from "@/components/SettingsCard";
import { useFetchApplicablePlays } from "@/hooks/PlayHooks";
import { useFetchPlayQueue } from "@/hooks/PlayQueueHooks";
import { useFetchTargetRuns } from "@/hooks/TrailHooks";
import { isQueued, type RunTab } from "@/models/Play";

interface TargetPlaysSectionProps {
  targetType: "ticket" | "doc";
  targetId: string;
  projectId: string;
  // A ticket's stage picks its plays; a doc has none.
  stage?: PlayStage | undefined;
  tab: RunTab;
}

// The plays this ticket or doc offers, the auto runs queued on it, and its runs, newest first; empty parts stay out.
export const TargetPlaysSection = ({ targetType, targetId, projectId, stage, tab }: TargetPlaysSectionProps) => {
  const { data: plays } = useFetchApplicablePlays(projectId, targetType, stage);
  const { data: queue } = useFetchPlayQueue(targetType, targetId);
  const { data: runs, error } = useFetchTargetRuns(targetType, targetId);
  const runnable = plays?.filter((play) => play.can_run) ?? [];
  const queued = queue?.items.filter(isQueued) ?? [];

  return (
    <View className="gap-5">
      {runnable.length > 0 && (
        <SettingsCard title="Plays">
          {runnable.map((play, i) => (
            <PlayStartRow key={play.id} play={play} targetType={targetType} targetId={targetId} projectId={projectId} tab={tab} first={i === 0} />
          ))}
        </SettingsCard>
      )}
      {queue?.paused && <AutoPlaysPausedRow targetType={targetType} targetId={targetId} projectId={projectId} autoRuns={queue.auto_runs} />}
      {queued.length > 0 && (
        <SettingsCard title="Queued">
          {queued.map((item, i) => (
            <QueuedRunRow key={item.id} item={item} title={item.play_label} resumable={false} first={i === 0} />
          ))}
        </SettingsCard>
      )}
      {error && <ErrorDisplay error={error} className="px-0" />}
      {runs && runs.length > 0 && (
        <SettingsCard title="Runs">
          {runs.map((run, i) => (
            <RunRow key={run.id} run={run} title={run.play_label} tab={tab} first={i === 0} />
          ))}
        </SettingsCard>
      )}
    </View>
  );
};
