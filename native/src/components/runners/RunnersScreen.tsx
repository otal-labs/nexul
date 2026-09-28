import { View } from "react-native";

import { RunnerFeed } from "@/components/runners/RunnerFeed";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { PlaceholderScreen } from "@/components/PlaceholderScreen";
import { useFetchRunners } from "@/hooks/RunnerHooks";

export const RunnersScreen = () => {
  const { data, error, isPending } = useFetchRunners();

  return (
    <View className="flex-1 bg-background">
      {isPending && <LoadingDisplay message="Loading runners…" />}
      {error && <ErrorDisplay error={error} />}
      {data && data.length === 0 && <PlaceholderScreen message="No runners yet." />}
      {data && data.length > 0 && <RunnerFeed runners={data} />}
    </View>
  );
};
