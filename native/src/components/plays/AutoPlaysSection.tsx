import { View } from "react-native";

import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { Microheader } from "@/components/Microheader";
import { AutoPlayRow } from "@/components/plays/AutoPlayRow";
import { SettingsCard } from "@/components/SettingsCard";
import { Text } from "@/components/ui/text";
import { useFetchAutoPlays } from "@/hooks/PlayHooks";
import { useAreaAccess } from "@/hooks/WorkspaceHooks";
import { subjectOf, type Play } from "@/models/Play";

// Readable and switchable here; adding one or changing when it starts stays on the web.
export const AutoPlaysSection = ({ play }: { play: Play }) => {
  const canRead = useAreaAccess()?.("autoPlays") === true;
  const { data: autoPlays, error } = useFetchAutoPlays(play.workspace_id, play.id, canRead);
  return (
    <View className="gap-2">
      {error && <ErrorDisplay error={error} className="px-0" />}
      {autoPlays && autoPlays.length === 0 && (
        <>
          <Microheader className="px-1">Auto plays</Microheader>
          <EmptyRow message="It never starts by itself" />
        </>
      )}
      {autoPlays && autoPlays.length > 0 && (
        <SettingsCard title="Auto plays">
          {autoPlays.map((autoPlay, i) => (
            <AutoPlayRow key={autoPlay.id} autoPlay={autoPlay} subject={subjectOf(play.type)} first={i === 0} />
          ))}
        </SettingsCard>
      )}
      {autoPlays && <Text className="px-1 text-row text-muted-foreground">Auto plays are added and edited on the web.</Text>}
    </View>
  );
};
