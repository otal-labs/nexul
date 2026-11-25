import { useState } from "react";
import { View } from "react-native";

import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { Microheader } from "@/components/Microheader";
import { RunRow } from "@/components/plays/RunRow";
import { SettingsCard } from "@/components/SettingsCard";
import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import { useFetchPlayRuns } from "@/hooks/TrailHooks";
import type { PlayRun } from "@/models/Trail";

const WAITING_SHOWN = 5;
const RECENT_PAGE = 10;
// shortcut: the list grows its one page up to the server's 100-row cap; reading past it needs offset pages.
const RECENT_MAX = 100;

const runTitle = (run: PlayRun) => [run.target_key, run.target_title].filter(Boolean).join(" · ") || run.play_label;

// The runs waiting on an answer, oldest first, over every run newest first; a waiting run is in both, as on the web.
export const PlayRunsSection = ({ playId }: { playId: string }) => {
  const [recentLimit, setRecentLimit] = useState(RECENT_PAGE);
  const { data: waiting } = useFetchPlayRuns(playId, true, WAITING_SHOWN);
  const { data: recent, error } = useFetchPlayRuns(playId, false, recentLimit);

  return (
    <View className="gap-5">
      {waiting && waiting.total > 0 && (
        <SettingsCard title={`Needs an answer · ${waiting.total}`}>
          {waiting.items.map((run, i) => (
            <RunRow key={run.id} run={run} title={runTitle(run)} tab="more" first={i === 0} />
          ))}
        </SettingsCard>
      )}
      {error && <ErrorDisplay error={error} className="px-0" />}
      {recent && recent.total === 0 && (
        <View className="gap-2">
          <Microheader className="px-1">Runs</Microheader>
          <EmptyRow message="No runs yet" />
        </View>
      )}
      {recent && recent.total > 0 && (
        <SettingsCard title={`Runs · ${recent.total}`}>
          {recent.items.map((run, i) => (
            <RunRow key={run.id} run={run} title={runTitle(run)} tab="more" first={i === 0} />
          ))}
        </SettingsCard>
      )}
      {recent && recent.total > recent.items.length && recentLimit < RECENT_MAX && (
        <Button variant="ghost" onPress={() => setRecentLimit(Math.min(recentLimit + RECENT_PAGE, RECENT_MAX))}>
          <Text>Show more</Text>
        </Button>
      )}
    </View>
  );
};
