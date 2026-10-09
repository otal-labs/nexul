import Server from "lucide-react-native/icons/server";
import { View } from "react-native";

import { EmptyState } from "@/components/EmptyState";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { RunnerFeed } from "@/components/runners/RunnerFeed";
import { ScreenHeader } from "@/components/ScreenHeader";
import { useFetchRunners } from "@/hooks/RunnerHooks";

export const RunnersScreen = () => {
  const { data, error, isPending } = useFetchRunners();

  return (
    <View className="flex-1 bg-background">
      {!(data && data.length > 0) && <ScreenHeader title="Runners" className="pt-2" />}
      {isPending && <LoadingDisplay message="Loading runners" />}
      {error && <ErrorDisplay error={error} />}
      {data && data.length === 0 && (
        <EmptyState icon={Server} title="No runners yet" message="A runner is added from a machine's page on the web; it shows here once it connects." />
      )}
      {data && data.length > 0 && <RunnerFeed runners={data} />}
    </View>
  );
};
