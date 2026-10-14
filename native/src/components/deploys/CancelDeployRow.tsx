import { useRouter } from "expo-router";
import { View } from "react-native";

import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import { useFetchStack } from "@/hooks/StackHooks";
import { useAreaAccess } from "@/hooks/WorkspaceHooks";
import { DeployStatus, type Deploy } from "@/models/Stack";

interface CancelDeployRowProps {
  deploy: Deploy;
}

// Offered while the deploy is waiting or running to whoever may deploy this stack; the sheet asks before it stops.
export const CancelDeployRow = ({ deploy }: CancelDeployRowProps) => {
  const router = useRouter();
  const { data: stack } = useFetchStack(deploy.stack_id);
  const canDeploy = useAreaAccess(stack?.project_id || undefined)?.("editDeploys") === true;
  const active = deploy.status === DeployStatus.Pending || deploy.status === DeployStatus.Running;
  if (!canDeploy || !active) return null;
  return (
    <View className="px-4 pb-3">
      <Button variant="outline" onPress={() => router.push({ pathname: "/deploys/cancel", params: { deployId: deploy.id } })}>
        <Text>Cancel deploy</Text>
      </Button>
    </View>
  );
};
