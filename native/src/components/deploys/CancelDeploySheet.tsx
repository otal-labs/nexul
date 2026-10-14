import { useLocalSearchParams, useRouter } from "expo-router";
import { View } from "react-native";

import { ActionButton } from "@/components/action/ActionButton";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { SheetTitle } from "@/components/SheetTitle";
import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import { useCancelDeploy, useFetchDeploy } from "@/hooks/DeployHooks";
import { useFetchStack } from "@/hooks/StackHooks";

export const CancelDeploySheet = () => {
  const router = useRouter();
  const { deployId } = useLocalSearchParams<{ deployId: string }>();
  const { data: deploy } = useFetchDeploy(deployId);
  const { data: stack } = useFetchStack(deploy?.stack_id);
  const cancel = useCancelDeploy();

  return (
    <View className="gap-4 bg-popover px-5 pb-6">
      <SheetTitle title={stack ? `Cancel the deploy of ${stack.name}?` : "Cancel this deploy?"} className="px-0 pb-0" />
      <Text className="text-[15px] leading-[22px] text-muted-foreground">
        The runner stops this deploy where it is. Start a new deploy from the stack when you want to try again.
      </Text>
      {cancel.error && <ErrorDisplay error={cancel.error} className="px-0" />}
      <ActionButton
        permission="deploys:write"
        projectId={stack?.project_id || undefined}
        verb="cancel deploys"
        name="Cancel deploy"
        variant="destructive"
        disabled={cancel.isPending}
        onPress={() => cancel.mutate(deployId, { onSuccess: () => router.back() })}
      >
        <Text>{cancel.isPending ? "Cancelling…" : "Cancel deploy"}</Text>
      </ActionButton>
      <Button variant="outline" disabled={cancel.isPending} onPress={() => router.back()}>
        <Text>Keep deploying</Text>
      </Button>
    </View>
  );
};
