import { FlatList } from "react-native";

import { RunnerRow } from "@/components/runners/RunnerRow";
import type { Runner } from "@/models/Runner";

interface RunnerFeedProps {
  runners: Runner[];
}

export const RunnerFeed = ({ runners }: RunnerFeedProps) => (
  <FlatList data={runners} keyExtractor={(runner) => runner.id} renderItem={({ item }) => <RunnerRow runner={item} />} />
);
