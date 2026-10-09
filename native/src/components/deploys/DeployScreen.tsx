import { LegendList } from "@legendapp/list/react-native";
import { Stack, useLocalSearchParams } from "expo-router";
import { View } from "react-native";

import { DeployLogLineRow } from "@/components/deploys/DeployLogLineRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { RelativeTime } from "@/components/RelativeTime";
import { ScreenHeader } from "@/components/ScreenHeader";
import { Text } from "@/components/ui/text";
import { useFetchDeploy, useFetchDeployLog } from "@/hooks/DeployHooks";
import { cn } from "@/lib/utils";
import { DeployStatus, deployStatusDot, deployTitle, type Deploy } from "@/models/Stack";

// The header names the outcome, the way the web's deploy page does.
const OUTCOME: Record<DeployStatus, string> = {
  [DeployStatus.Pending]: "Waiting for a runner",
  [DeployStatus.Running]: "Deploying",
  [DeployStatus.Healthy]: "Deployed",
  [DeployStatus.Failed]: "Deploy failed",
};

const DeployMeta = ({ deploy }: { deploy: Deploy }) => (
  <>
    <View className={cn("size-2 rounded-full", deployStatusDot(deploy.status))} />
    <Text className="font-mono text-xs text-muted-foreground">{deployTitle(deploy)}</Text>
    <Text className="font-mono text-xs text-muted-foreground">
      <RelativeTime iso={deploy.created_at} />
    </Text>
  </>
);

export const DeployScreen = () => {
  const { id } = useLocalSearchParams<{ id: string }>();
  const { data: deploy, error: deployError, isPending: deployPending } = useFetchDeploy(id);
  const { data: lines, error: logError, isPending: logPending } = useFetchDeployLog(id);

  return (
    <View className="flex-1 bg-background">
      <Stack.Screen options={{ title: "Deploy" }} />
      {deployPending && <LoadingDisplay message="Loading the deploy" />}
      {deployError && <ErrorDisplay error={deployError} notFound="This deploy doesn't exist or was deleted." />}
      {deploy && <ScreenHeader title={OUTCOME[deploy.status] ?? deploy.status} meta={<DeployMeta deploy={deploy} />} className="pt-2" />}
      {logPending && <LoadingDisplay message="Loading the log" />}
      {logError && <ErrorDisplay error={logError} />}
      {lines && lines.length === 0 && (
        <View className="mx-4 mb-4 flex-1 items-center justify-center rounded-xl border border-border bg-surface-2 px-6">
          <Text className="text-center text-[13px] text-muted-foreground">Waiting for the runner to pick this up</Text>
        </View>
      )}
      {lines && lines.length > 0 && (
        <View className="mx-3 mb-3 flex-1 overflow-hidden rounded-xl border border-border bg-surface-2">
          <LegendList
            data={lines}
            keyExtractor={(line) => String(line.seq)}
            renderItem={({ item }) => <DeployLogLineRow line={item} />}
            estimatedItemSize={20}
            recycleItems
            initialScrollAtEnd
            alignItemsAtEnd
            maintainScrollAtEnd
            contentContainerStyle={{ paddingVertical: 8 }}
          />
        </View>
      )}
    </View>
  );
};
