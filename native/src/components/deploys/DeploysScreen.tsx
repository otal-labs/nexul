import { useRouter } from "expo-router";
import Rocket from "lucide-react-native/icons/rocket";

import { StackFeed } from "@/components/deploys/StackFeed";
import { EmptyState } from "@/components/EmptyState";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { HandOff, useLoaderShown } from "@/components/HandOff";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ScreenHeader } from "@/components/ScreenHeader";
import { FieldScreen } from "@/components/FieldScreen";
import { useFetchStacksWithLatestDeploy } from "@/hooks/StackHooks";
import { useSelectedWorkspace } from "@/hooks/WorkspaceHooks";

export const DeploysScreen = () => {
  const router = useRouter();
  const workspace = useSelectedWorkspace();
  const { data, error, isPending } = useFetchStacksWithLatestDeploy();
  const waited = useLoaderShown(isPending);

  return (
    <FieldScreen>
      {!(data && data.length > 0) && <ScreenHeader eyebrow={workspace?.name} title="Deploys" />}
      {isPending && <LoadingDisplay message="Loading stacks" />}
      {error && <ErrorDisplay error={error} />}
      {data && data.length === 0 && (
        <EmptyState icon={Rocket} title="Nothing deployed yet" message="Stacks set up on the web show their deploys here." />
      )}
      {data && data.length > 0 && (
        <HandOff after={waited}>
          <StackFeed rows={data} onSelect={(id) => router.push(`/deploys/stack/${id}`)} />
        </HandOff>
      )}
    </FieldScreen>
  );
};
