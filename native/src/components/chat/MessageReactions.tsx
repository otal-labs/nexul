import { View } from "react-native";

import { Text } from "@/components/ui/text";
import type { Reaction } from "@/models/Chat";

// Reactions read here as emoji and count; adding one happens on the web, phones type emoji from their keyboard.
export const MessageReactions = ({ reactions }: { reactions: Reaction[] }) => (
  <View className="flex-row flex-wrap gap-1 pt-1">
    {reactions.map((r) => (
      <View
        key={r.emoji}
        accessible
        accessibilityLabel={`${r.emoji} ${r.user_ids.length}`}
        className="h-6 flex-row items-center gap-1 rounded-full border border-border bg-card px-2"
      >
        <Text className="text-xs">{r.emoji}</Text>
        <Text className="font-mono text-xs text-muted-foreground">{r.user_ids.length}</Text>
      </View>
    ))}
  </View>
);
