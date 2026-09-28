import { View } from "react-native";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { DevicesFeed } from "@/components/settings/DevicesFeed";
import { useListSessions } from "@/hooks/SessionHooks";

export const DevicesScreen = () => {
  const { data, error, isPending } = useListSessions();

  return (
    <View className="flex-1 bg-background">
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {data && <DevicesFeed sessions={data.sessions} />}
    </View>
  );
};
