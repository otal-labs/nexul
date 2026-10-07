import { useState } from "react";
import { View } from "react-native";

import { DiscordMarkdown } from "@/components/chat/DiscordMarkdown";
import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";
import type { EmbedField } from "@/models/Embed";

// Below this width a 40% label column wraps every word, so each label sits over its value instead.
const STACK_BELOW = 280;

const EmbedFieldRow = ({ field, first, stacked }: { field: EmbedField; first: boolean; stacked: boolean }) => (
  <View className={cn("border-cell-line", !stacked && "flex-row", !first && "border-t")}>
    <View className={cn("border-cell-line bg-cell-label px-2.5 py-1.5", !stacked && "w-2/5 border-r")}>
      <Text className="text-xs text-muted-foreground">{field.name}</Text>
    </View>
    <View className="min-w-0 flex-1 px-2.5 py-1.5">
      <DiscordMarkdown text={field.value} textClassName="text-xs leading-4 text-foreground/90" />
    </View>
  </View>
);

// Every field is a label and value row, inline or not, framed with a hairline on every cell edge.
export const EmbedFieldGrid = ({ fields }: { fields: EmbedField[] }) => {
  const [stacked, setStacked] = useState(false);
  return (
    <View
      onLayout={({ nativeEvent }) => setStacked(nativeEvent.layout.width < STACK_BELOW)}
      className="overflow-hidden rounded-md border border-cell-line"
    >
      {fields.map((field, i) => (
        <EmbedFieldRow key={i} field={field} first={i === 0} stacked={stacked} />
      ))}
    </View>
  );
};
