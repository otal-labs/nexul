import { useLocalSearchParams, useRouter } from "expo-router";
import { View } from "react-native";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { SheetTitle } from "@/components/SheetTitle";
import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import { useDeployStack, useFetchStack } from "@/hooks/StackHooks";

type RedeployParams = { stackId: string; image: string };

export const RedeployScreen = () => {
  const router = useRouter();
  const { stackId, image } = useLocalSearchParams<RedeployParams>();
  const { data: stack } = useFetchStack(stackId);
  const deploy = useDeployStack();

  return (
    <View className="gap-4 bg-popover px-4 pb-6">
      <SheetTitle title={stack ? `Redeploy ${stack.name}` : "Redeploy"} className="px-0 pb-0" />
      <Text variant="muted">Pulls the image again and restarts the container.</Text>
      <Text className="font-mono text-xs" numberOfLines={1}>
        {image}
      </Text>
      {deploy.error && <ErrorDisplay error={deploy.error} className="px-0" />}
      <Button
        disabled={deploy.isPending}
        onPress={() => deploy.mutate({ stackId, image }, { onSuccess: () => router.back() })}
      >
        <Text>{deploy.isPending ? "Redeploying…" : "Redeploy"}</Text>
      </Button>
      <Button variant="outline" disabled={deploy.isPending} onPress={() => router.back()}>
        <Text>Cancel</Text>
      </Button>
    </View>
  );
};
