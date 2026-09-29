import { LegendList } from "@legendapp/list/react-native";
import { Stack, useLocalSearchParams } from "expo-router";
import { View } from "react-native";

import { DeployLogLineRow } from "@/components/deploys/DeployLogLineRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Text } from "@/components/ui/text";
import { useFetchDeploy, useFetchDeployLog } from "@/hooks/DeployHooks";
import { cn } from "@/lib/utils";
import { deployStatusDot, deployTitle } from "@/models/Stack";

export const DeployScreen = () => {
  const { id } = useLocalSearchParams<{ id: string }>();
  const { data: deploy, error: deployError, isPending: deployPending } = useFetchDeploy(id);
  const { data: lines, error: logError, isPending: logPending } = useFetchDeployLog(id);

  return (
    <View className="flex-1 bg-background">
      {deployPending && <LoadingDisplay />}
      {deployError && <ErrorDisplay error={deployError} notFound="This deploy doesn't exist or was deleted." />}
      {deploy && <Stack.Screen options={{ title: deployTitle(deploy) }} />}
      {deploy && (
        <View className="flex-row items-center gap-2 border-b border-border px-4 py-3">
          <View className={cn("size-2 rounded-full", deployStatusDot(deploy.status))} />
          <Text className="min-w-0 flex-1 font-mono text-xs" numberOfLines={1}>
            {deploy.image || "repo build"}
          </Text>
          <Text variant="small" className="text-muted-foreground">
            {deploy.status}
          </Text>
        </View>
      )}

      {logPending && <LoadingDisplay message="Loading the log…" />}
      {logError && <ErrorDisplay error={logError} />}
      {lines && lines.length === 0 && (
        <View className="flex-1 items-center justify-center px-6">
          <Text variant="muted" className="text-center">
            Waiting for the runner to pick this up…
          </Text>
        </View>
      )}
      {lines && lines.length > 0 && (
        <View className="flex-1 bg-surface-2">
          <LegendList
            data={lines}
            keyExtractor={(line) => String(line.seq)}
            renderItem={({ item }) => <DeployLogLineRow line={item} />}
            estimatedItemSize={20}
            recycleItems
            initialScrollAtEnd
            alignItemsAtEnd
            maintainScrollAtEnd
          />
        </View>
      )}
    </View>
  );
};
