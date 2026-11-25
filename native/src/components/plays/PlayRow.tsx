import { useRouter } from "expo-router";
import { memo } from "react";
import { Pressable, View } from "react-native";

import { PlayEnabledSwitch } from "@/components/plays/PlayEnabledSwitch";
import { PlayHealthLine } from "@/components/plays/PlayHealthLine";
import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";
import type { WorkspacePlay } from "@/models/Play";

// The play's name over what its runs say, its switch trailing; a play switched off reads muted.
export const PlayRow = memo(({ play }: { play: WorkspacePlay }) => {
  const router = useRouter();
  return (
    <View className="min-h-16 flex-row items-center gap-3 pr-4">
      <Pressable role="button" aria-label={play.label} onPress={() => router.push(`/more/plays/${play.id}`)} className="min-w-0 flex-1 gap-0.5 py-3 pl-5 active:bg-accent">
        <Text numberOfLines={1} className={cn("text-body font-medium", !play.enabled && "text-muted-foreground")}>
          {play.label}
        </Text>
        <PlayHealthLine health={play} />
      </Pressable>
      <PlayEnabledSwitch play={play} />
    </View>
  );
});
PlayRow.displayName = "PlayRow";
