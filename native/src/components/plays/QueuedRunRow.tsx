import { View } from "react-native";

import { waitingLines } from "@nexul/client-core/playQueue";

import { ActionButton } from "@/components/action/ActionButton";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { Text } from "@/components/ui/text";
import { useCancelQueuedRun } from "@/hooks/PlayQueueHooks";
import { cn } from "@/lib/utils";
import type { PlayQueueItem } from "@/models/Play";

interface QueuedRunRowProps {
  item: PlayQueueItem;
  // Names the play on a ticket's list, where the ticket is already the screen.
  title: string;
  first: boolean;
}

// Queued, then why it waits in the web's words: "Queued · their computer is offline".
const waitingText = (item: PlayQueueItem) => `Queued · ${waitingLines(item).join(", ")}`;

// An auto play's run that hasn't started: Cancel takes it out.
export const QueuedRunRow = ({ item, title, first }: QueuedRunRowProps) => {
  const cancel = useCancelQueuedRun();
  return (
    <View className={cn("gap-2 px-4 py-2.5", !first && "border-t border-border")}>
      <View className="min-h-8 flex-row items-center gap-3">
        <View className="size-2 rounded-full border border-muted-foreground" />
        <View className="min-w-0 flex-1 gap-0.5">
          <Text numberOfLines={1} className="text-body">
            {title}
          </Text>
          <Text numberOfLines={2} className="text-row text-muted-foreground">
            {waitingText(item)}
          </Text>
        </View>
        <ActionButton
          permission="autoplays:write"
          projectId={item.project_id}
          verb="cancel queued runs"
          name={`Cancel ${item.play_label}`}
          aria-label={`Cancel ${item.play_label}`}
          variant="ghost"
          size="sm"
          disabled={cancel.isPending}
          onPress={() => cancel.mutate(item.id)}
        >
          <Text>Cancel</Text>
        </ActionButton>
      </View>
      {cancel.error && <ErrorDisplay error={cancel.error} className="px-0" />}
    </View>
  );
};
