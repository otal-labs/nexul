import { useRouter } from "expo-router";
import { View } from "react-native";

import { ActionButton } from "@/components/action/ActionButton";
import { Text } from "@/components/ui/text";
import { useFetchStack } from "@/hooks/StackHooks";
import { DeployStatus, type Deploy } from "@/models/Stack";

interface CancelDeployRowProps {
  deploy: Deploy;
}

// Offered while the deploy is waiting or running; the sheet asks before it stops.
export const CancelDeployRow = ({ deploy }: CancelDeployRowProps) => {
  const router = useRouter();
  const { data: stack } = useFetchStack(deploy.stack_id);
  const active = deploy.status === DeployStatus.Pending || deploy.status === DeployStatus.Running;
  if (!active) return null;
  return (
    <View className="px-4 pb-3">
      <ActionButton
        permission="deploys:write"
        projectId={stack?.project_id || undefined}
        verb="cancel deploys"
        name="Cancel deploy"
        variant="outline"
        onPress={() => router.push({ pathname: "/deploys/cancel", params: { deployId: deploy.id } })}
      >
        <Text>Cancel deploy</Text>
      </ActionButton>
    </View>
  );
};
