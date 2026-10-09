import { FlatList } from "react-native";

import { RunnerRow } from "@/components/runners/RunnerRow";
import { ScreenHeader } from "@/components/ScreenHeader";
import type { Runner } from "@/models/Runner";

interface RunnerFeedProps {
  runners: Runner[];
}

export const RunnerFeed = ({ runners }: RunnerFeedProps) => {
  const online = runners.filter((r) => r.connected).length;
  return (
    <FlatList
      data={runners}
      keyExtractor={(runner) => runner.id}
      ListHeaderComponent={<ScreenHeader title="Runners" meta={`${online} of ${runners.length} online`} className="pt-2" />}
      contentContainerClassName="pb-6"
      renderItem={({ item }) => <RunnerRow runner={item} />}
    />
  );
};
