import { Stack, useLocalSearchParams, useRouter } from "expo-router";
import { ScrollView, View } from "react-native";

import { ContainerRow } from "@/components/deploys/ContainerRow";
import { DeployRow } from "@/components/deploys/DeployRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import { useFetchStack, useFetchStackDeploys, useFetchStackServices } from "@/hooks/StackHooks";
import { useAreaAccess } from "@/hooks/WorkspaceHooks";
import { DeployStatus, DeployStrategy, latestDeploy } from "@/models/Stack";

export const StackScreen = () => {
  const router = useRouter();
  const { id } = useLocalSearchParams<{ id: string }>();
  const { data: stack, error, isPending } = useFetchStack(id);
  const { data: deploys, error: deploysError, isPending: deploysPending } = useFetchStackDeploys(id);
  const { data: containers, error: containersError, isPending: containersPending } = useFetchStackServices(id);
  const canReadLogs = useAreaAccess()?.("stackLogs") === true;

  const logsOpener = (stackId: string, service: string) => {
    if (!canReadLogs) return undefined;
    return () => router.push({ pathname: "/deploys/stack/[id]/logs/[service]", params: { id: stackId, service } });
  };

  const latest = latestDeploy(deploys);
  const isRun = stack?.strategy === DeployStrategy.Run;
  const canRedeploy = isRun && !!latest?.image;
  // The server refuses a second deploy while one is active; live deploy.updated events re-enable the button.
  const deploying = latest?.status === DeployStatus.Pending || latest?.status === DeployStatus.Running;

  return (
    <ScrollView className="flex-1 bg-background">
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} notFound="This stack doesn't exist or was deleted." />}
      {stack && (
        <View className="gap-5 px-4 py-4">
          <Stack.Screen options={{ title: stack.name }} />
          <View>
            <Text variant="h3">{stack.name}</Text>
            <Text variant="muted" className="font-mono text-xs">
              {stack.machine}
            </Text>
          </View>

          {canRedeploy && latest && (
            <Button
              disabled={deploying}
              onPress={() =>
                router.push({ pathname: "/deploys/redeploy", params: { stackId: stack.id, image: latest.image } })
              }
            >
              <Text>{deploying ? "Deploy in progress" : "Redeploy"}</Text>
            </Button>
          )}
          {stack.strategy === DeployStrategy.Compose && (
            <Text variant="muted" className="text-xs">
              Redeploy from the web: this stack builds from its compose file.
            </Text>
          )}
          {isRun && !canRedeploy && (
            <Text variant="muted" className="text-xs">
              Nothing to redeploy yet: this stack has no deployed image.
            </Text>
          )}

          <View className="gap-2 border-t border-border pt-3">
            <Text variant="small" className="text-muted-foreground">
              Services
            </Text>
            {containersPending && <LoadingDisplay />}
            {containersError && <ErrorDisplay error={containersError} className="px-0" />}
            {containers && containers.length === 0 && <Text variant="muted">No services parsed yet.</Text>}
            {containers &&
              containers.length > 0 &&
              containers.map((c) => <ContainerRow key={c.id} container={c} onOpenLogs={logsOpener(stack.id, c.name)} />)}
          </View>

          <View className="gap-2 border-t border-border pt-3">
            <Text variant="small" className="text-muted-foreground">
              Deploy history
            </Text>
            {deploysPending && <LoadingDisplay />}
            {deploysError && <ErrorDisplay error={deploysError} className="px-0" />}
            {deploys && deploys.length === 0 && <Text variant="muted">No deploys yet.</Text>}
            {deploys &&
              deploys.length > 0 &&
              deploys.map((d) => <DeployRow key={d.id} deploy={d} onPress={() => router.push(`/deploys/deploy/${d.id}`)} />)}
          </View>
        </View>
      )}
    </ScrollView>
  );
};
