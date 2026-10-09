import { FileText, Lock } from "lucide-react-native";
import { Pressable, View } from "react-native";
import { useCSSVariable } from "uniwind";

import { RelativeTime } from "@/components/RelativeTime";
import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";
import type { DocListItem } from "@/models/Doc";

interface DocRowProps {
  doc: DocListItem;
  onPress: (doc: DocListItem) => void;
}

// A doc the viewer can't open keeps full contrast and says so with a lock, rather than fading out.
export const DocRow = ({ doc, onPress }: DocRowProps) => {
  const [mutedForeground] = useCSSVariable(["--color-muted-foreground"]);
  return (
    <Pressable
      role="button"
      disabled={!doc.can_open}
      onPress={() => onPress(doc)}
      className="min-h-14 flex-row items-center gap-3.5 px-5 py-2.5 active:bg-accent"
    >
      <FileText size={18} color={String(mutedForeground)} />
      <View className="min-w-0 flex-1 gap-0.5">
        <Text numberOfLines={2} className={cn("text-[15px] leading-5", !doc.can_open && "text-muted-foreground")}>
          {doc.title}
        </Text>
        <Text className="font-mono text-xs text-muted-foreground">
          v{doc.version} · <RelativeTime iso={doc.updated_at} />
        </Text>
      </View>
      {!doc.can_open && <Lock accessibilityLabel="Locked" size={14} color={String(mutedForeground)} />}
    </Pressable>
  );
};
