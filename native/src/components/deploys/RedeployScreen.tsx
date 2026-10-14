import { useLocalSearchParams, useRouter } from "expo-router";
import { View } from "react-native";

import { ActionButton } from "@/components/action/ActionButton";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { SheetTitle } from "@/components/SheetTitle";
import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import { useDeployStack, useFetchStack, useStackActions } from "@/hooks/StackHooks";

type RedeployParams = { stackId: string; image: string };

export const RedeployScreen = () => {
  const router = useRouter();
  const { stackId, image } = useLocalSearchParams<RedeployParams>();
  const { data: stack } = useFetchStack(stackId);
  const answer = useStackActions(stack);
  const deploy = useDeployStack();

  return (
    <View className="gap-4 bg-popover px-5 pb-6">
      <SheetTitle title={stack ? `Redeploy ${stack.name}` : "Redeploy"} className="px-0 pb-0" />
      <Text className="text-[15px] leading-[22px] text-muted-foreground">Pulls the image again and restarts the container.</Text>
      <Text className="rounded-md bg-surface-2 px-3 py-2.5 font-mono text-xs" numberOfLines={1}>
        {image}
      </Text>
      {deploy.error && <ErrorDisplay error={deploy.error} className="px-0" />}
      <ActionButton
        permission="deploys:write"
        projectId={stack?.project_id || undefined}
        answer={answer("deploys:write")}
        verb="deploy"
        name="Redeploy"
        disabled={deploy.isPending}
        onPress={() => deploy.mutate({ stackId, image }, { onSuccess: () => router.back() })}
      >
        <Text>{deploy.isPending ? "Redeploying…" : "Redeploy"}</Text>
      </ActionButton>
      <Button variant="outline" disabled={deploy.isPending} onPress={() => router.back()}>
        <Text>Cancel</Text>
      </Button>
    </View>
  );
};
