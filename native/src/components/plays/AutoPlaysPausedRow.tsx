import Pause from "lucide-react-native/icons/pause";
import { View } from "react-native";

import { ActionButton } from "@/components/action/ActionButton";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { Icon } from "@/components/ui/icon";
import { Text } from "@/components/ui/text";
import { useResumeAutoPlays } from "@/hooks/PlayQueueHooks";

interface AutoPlaysPausedRowProps {
  targetType: "ticket" | "doc";
  targetId: string;
  projectId: string;
  autoRuns: number;
}

// The daily cap on automatic runs stopped this ticket's auto plays until someone resumes them.
export const AutoPlaysPausedRow = ({ targetType, targetId, projectId, autoRuns }: AutoPlaysPausedRowProps) => {
  const resume = useResumeAutoPlays();
  return (
    <View className="gap-2 rounded-xl border border-border bg-card px-4 py-2.5">
      <View className="min-h-8 flex-row items-center gap-3">
        <Icon as={Pause} className="size-4 text-warning" />
        <View className="min-w-0 flex-1 gap-0.5">
          <Text className="text-body">Auto plays paused</Text>
          <Text className="font-mono text-xs text-muted-foreground">
            {autoRuns} {autoRuns === 1 ? "run" : "runs"} today
          </Text>
        </View>
        <ActionButton
          permission="autoplays:write"
          projectId={projectId}
          verb="resume auto plays"
          name="Resume auto plays"
          variant="outline"
          size="sm"
          disabled={resume.isPending}
          onPress={() => resume.mutate({ targetType, targetId })}
        >
          <Text>Resume</Text>
        </ActionButton>
      </View>
      {resume.error && <ErrorDisplay error={resume.error} className="px-0" />}
    </View>
  );
};
