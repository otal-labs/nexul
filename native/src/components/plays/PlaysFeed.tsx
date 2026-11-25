import { LegendList } from "@legendapp/list/react-native";

import { Microheader } from "@/components/Microheader";
import { NowSection } from "@/components/plays/NowSection";
import { PlayRow } from "@/components/plays/PlayRow";
import { ScreenHeader } from "@/components/ScreenHeader";
import { Text } from "@/components/ui/text";
import { useSelectedWorkspace } from "@/hooks/WorkspaceHooks";
import type { WorkspacePlay } from "@/models/Play";

// One list: the header and what runs now above it, then a row per play, mounted as it scrolls into view.
export const PlaysFeed = ({ plays }: { plays: WorkspacePlay[] }) => {
  const workspace = useSelectedWorkspace();
  return (
    <LegendList
      className="flex-1"
      data={plays}
      keyExtractor={(play) => play.id}
      renderItem={({ item }) => <PlayRow play={item} />}
      estimatedItemSize={64}
      contentContainerClassName="pb-8"
      ListHeaderComponent={
        <>
          <ScreenHeader eyebrow={workspace?.name} title="Plays" className="pt-2" />
          <NowSection plays={plays} />
          <Microheader className="px-5 pb-1">Plays</Microheader>
        </>
      }
      ListFooterComponent={<Text className="px-5 pt-4 text-row text-muted-foreground">New plays and their auto plays are set up on the web.</Text>}
    />
  );
};
