import { View } from "react-native";

import { fieldTone, type EmbedField } from "@nexul/client-core/embed";

import { DiscordMarkdown } from "@/components/chat/DiscordMarkdown";
import { toneBg } from "@/components/chat/EmbedTone";
import { Microheader } from "@/components/Microheader";
import { cn } from "@/lib/utils";

// An inline value past this many characters (an image tag, a URL) takes the row instead of wrapping in a half.
const LONG_VALUE = 24;

const EmbedFieldItem = ({ field }: { field: EmbedField }) => {
  const tone = fieldTone(field.value);
  const wide = !field.inline || field.value.length > LONG_VALUE;
  return (
    <View className={cn("gap-0.5", wide ? "w-full" : "w-[47%]")}>
      <Microheader>{field.name}</Microheader>
      <View className="flex-row items-center gap-1.5">
        {tone && <View className={cn("size-1.5 rounded-full", toneBg[tone])} />}
        <View className="min-w-0 flex-1">
          <DiscordMarkdown text={field.value} textClassName="text-[13px] leading-[18px] text-foreground" />
        </View>
      </View>
    </View>
  );
};

// Facts, not a table: a mono label over each value, the sender's inline fields two across and the rest the full width.
export const EmbedFieldGrid = ({ fields }: { fields: EmbedField[] }) => (
  <View className="flex-row flex-wrap gap-x-[6%] gap-y-3">
    {fields.map((field, i) => (
      <EmbedFieldItem key={i} field={field} />
    ))}
  </View>
);
