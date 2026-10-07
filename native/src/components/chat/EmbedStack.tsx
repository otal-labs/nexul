import { useState } from "react";
import { View } from "react-native";

import { EmbedCard } from "@/components/chat/EmbedCard";
import { FoldBar } from "@/components/chat/FoldBar";
import { EMBED_STACK_LIMIT, moreEmbedsLabel, type Embed } from "@/models/Embed";

// Past two embeds a post folds the rest behind one bar.
export const EmbedStack = ({ embeds }: { embeds: Embed[] }) => {
  const [open, setOpen] = useState(false);
  const shown = open ? embeds : embeds.slice(0, EMBED_STACK_LIMIT);
  const hidden = embeds.length - EMBED_STACK_LIMIT;
  return (
    <View className="gap-2.5 pt-1">
      {shown.map((embed, i) => (
        <EmbedCard key={i} embed={embed} />
      ))}
      {hidden > 0 && <FoldBar label={open ? "Show less" : moreEmbedsLabel(hidden)} open={open} onToggle={() => setOpen(!open)} />}
    </View>
  );
};
