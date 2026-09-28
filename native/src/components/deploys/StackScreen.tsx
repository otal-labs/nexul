import { useLocalSearchParams, useRouter } from "expo-router";
import { ScrollView, View } from "react-native";

import { ContainerRow } from "@/components/deploys/ContainerRow";
import { DeployRow } from "@/components/deploys/DeployRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import { useFetchStack, useFetchStackDeploys, useFetchStackServices } from "@/hooks/StackHooks";
import { DeployStrategy, latestDeploy } from "@/models/Stack";

export const StackScreen = () => {
  const router = useRouter();
  const { id } = useLocalSearchParams<{ id: string }>();
  const { data: stack, error, isPending } = useFetchStack(id);
  const { data: deploys, error: deploysError, isPending: deploysPending } = useFetchStackDeploys(id);
  const { data: containers, error: containersError, isPending: containersPending } = useFetchStackServices(id);

  const latest = latestDeploy(deploys);
  const canRedeploy = stack?.strategy === DeployStrategy.Run && !!latest?.image;

  return (
    <ScrollView className="flex-1 bg-background">
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {stack && (
        <View className="gap-5 px-4 py-4">
          <View>
            <Text variant="h3">{stack.name}</Text>
            <Text variant="muted" className="font-mono text-xs">
              {stack.machine}
            </Text>
          </View>

          {canRedeploy && latest && (
            <Button
              onPress={() =>
                router.push({ pathname: "/deploys/redeploy", params: { stackId: stack.id, image: latest.image } })
              }
            >
              <Text>Redeploy</Text>
            </Button>
          )}
          {!canRedeploy && (
            <Text variant="muted" className="text-xs">
              Redeploy from the phone needs a running image; build stacks deploy from the web.
            </Text>
          )}

          <View className="gap-2 border-t border-border pt-3">
            <Text variant="small" className="text-muted-foreground">
              Services
            </Text>
            {containersPending && <LoadingDisplay />}
            {containersError && <ErrorDisplay error={containersError} />}
            {containers && containers.length === 0 && <Text variant="muted">No services parsed yet.</Text>}
            {containers && containers.length > 0 && containers.map((c) => <ContainerRow key={c.id} container={c} />)}
          </View>

          <View className="gap-2 border-t border-border pt-3">
            <Text variant="small" className="text-muted-foreground">
              Deploy history
            </Text>
            {deploysPending && <LoadingDisplay />}
            {deploysError && <ErrorDisplay error={deploysError} />}
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
