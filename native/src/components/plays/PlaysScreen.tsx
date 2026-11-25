import Zap from "lucide-react-native/icons/zap";
import { View } from "react-native";

import { AreaGate } from "@/components/AreaGate";
import { EmptyState } from "@/components/EmptyState";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { PlaysFeed } from "@/components/plays/PlaysFeed";
import { ScreenHeader } from "@/components/ScreenHeader";
import { useFetchWorkspacePlays } from "@/hooks/PlayHooks";

// Every play the viewer sees, what runs and waits now across them, each play's health and switch.
export const PlaysScreen = () => {
  const { data, error, isPending } = useFetchWorkspacePlays();
  return (
    <AreaGate area="plays">
      <View className="flex-1 bg-background">
        {!(data && data.length > 0) && <ScreenHeader title="Plays" className="pt-2" />}
        {isPending && <LoadingDisplay message="Loading plays" />}
        {error && <ErrorDisplay error={error} />}
        {data && data.length === 0 && <EmptyState icon={Zap} title="No plays yet" message="Plays set up on the web show here with their runs." />}
        {data && data.length > 0 && <PlaysFeed plays={data} />}
      </View>
    </AreaGate>
  );
};
