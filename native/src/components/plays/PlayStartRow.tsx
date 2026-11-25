import { useRouter } from "expo-router";
import { View } from "react-native";

import { ActionButton } from "@/components/action/ActionButton";
import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";
import { runSheetRoute, type ListedPlay, type PlayType, type RunTab } from "@/models/Play";

interface PlayStartRowProps {
  play: ListedPlay;
  targetType: PlayType;
  targetId: string;
  projectId: string;
  tab: RunTab;
  first: boolean;
}

// A play this ticket or doc offers, with Run opening the sheet that asks for this run's instructions.
export const PlayStartRow = ({ play, targetType, targetId, projectId, tab, first }: PlayStartRowProps) => {
  const router = useRouter();
  return (
    <View className={cn("min-h-[52px] flex-row items-center gap-3 px-4 py-2.5", !first && "border-t border-border")}>
      <View className="min-w-0 flex-1 gap-0.5">
        <Text numberOfLines={1} className="text-body">
          {play.label}
        </Text>
        {play.description !== "" && (
          <Text numberOfLines={2} className="text-row text-muted-foreground">
            {play.description}
          </Text>
        )}
      </View>
      <ActionButton
        permission="plays:run"
        projectId={projectId}
        verb="run plays"
        name={`Run ${play.label}`}
        aria-label={`Run ${play.label}`}
        variant="outline"
        size="sm"
        onPress={() => router.push(runSheetRoute(tab, play, targetType, targetId, projectId))}
      >
        <Text>Run</Text>
      </ActionButton>
    </View>
  );
};
