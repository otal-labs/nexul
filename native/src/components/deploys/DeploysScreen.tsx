import { useRouter } from "expo-router";
import { View } from "react-native";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { StackFeed } from "@/components/deploys/StackFeed";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { PlaceholderScreen } from "@/components/PlaceholderScreen";
import { useFetchStacksWithLatestDeploy } from "@/hooks/StackHooks";

export const DeploysScreen = () => {
  const router = useRouter();
  const { data, error, isPending } = useFetchStacksWithLatestDeploy();

  return (
    <View className="flex-1 bg-background">
      {isPending && <LoadingDisplay message="Loading stacks…" />}
      {error && <ErrorDisplay error={error} />}
      {data && data.length === 0 && <PlaceholderScreen message="No stacks yet." />}
      {data && data.length > 0 && (
        <StackFeed rows={data} onSelect={(id) => router.push(`/deploys/stack/${id}`)} />
      )}
    </View>
  );
};
