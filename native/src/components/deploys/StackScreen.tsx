import { Stack, useLocalSearchParams, useRouter } from "expo-router";
import { ScrollView, View } from "react-native";

import { ContainerRow } from "@/components/deploys/ContainerRow";
import { DeployRow } from "@/components/deploys/DeployRow";
import { StackLiveWell } from "@/components/deploys/StackLiveWell";
import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Microheader } from "@/components/Microheader";
import { ScreenHeader } from "@/components/ScreenHeader";
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
  const canReadLogs = useAreaAccess(stack?.project_id || undefined)?.("stackLogs") === true;

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
    <ScrollView className="flex-1 bg-background" contentContainerClassName="pb-8">
      <Stack.Screen options={{ title: "Stack" }} />
      {isPending && <LoadingDisplay message="Loading the stack" />}
      {error && <ErrorDisplay error={error} notFound="This stack doesn't exist or was deleted." />}
      {stack && <ScreenHeader eyebrow={stack.machine} title={stack.name} meta={`${stack.strategy} stack`} className="pt-2" />}
      {stack && (
        <View className="gap-6 px-4">
          <StackLiveWell latest={latest} containers={containers} />

          {canRedeploy && latest && (
            <Button
              disabled={deploying}
              onPress={() => router.push({ pathname: "/deploys/redeploy", params: { stackId: stack.id, image: latest.image } })}
            >
              <Text>{deploying ? "Deploy in progress" : "Redeploy"}</Text>
            </Button>
          )}
          {stack.strategy === DeployStrategy.Compose && (
            <Text className="text-[13px] text-muted-foreground">Redeploy from the web: this stack builds from its compose file.</Text>
          )}
          {isRun && !canRedeploy && <Text className="text-[13px] text-muted-foreground">Nothing to redeploy yet: this stack has no deployed image.</Text>}

          <View className="gap-2">
            <Microheader className="px-1">Services</Microheader>
            {containersPending && <LoadingDisplay message="Loading services" />}
            {containersError && <ErrorDisplay error={containersError} className="px-0" />}
            {containers && containers.length === 0 && <EmptyRow message="No services parsed yet." />}
            {containers && containers.length > 0 && (
              <View className="overflow-hidden rounded-xl border border-border bg-card">
                {containers.map((c, i) => (
                  <ContainerRow key={c.id} container={c} first={i === 0} onOpenLogs={logsOpener(stack.id, c.name)} />
                ))}
              </View>
            )}
          </View>

          <View className="gap-2">
            <Microheader className="px-1">Deploy history</Microheader>
            {deploysPending && <LoadingDisplay message="Loading deploys" />}
            {deploysError && <ErrorDisplay error={deploysError} className="px-0" />}
            {deploys && deploys.length === 0 && <EmptyRow message="No deploys yet." />}
            {deploys && deploys.length > 0 && (
              <View className="overflow-hidden rounded-xl border border-border bg-card">
                {deploys.map((d, i) => (
                  <DeployRow key={d.id} deploy={d} first={i === 0} onPress={() => router.push(`/deploys/deploy/${d.id}`)} />
                ))}
              </View>
            )}
          </View>
        </View>
      )}
    </ScrollView>
  );
};
