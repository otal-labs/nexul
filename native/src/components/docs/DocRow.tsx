import FileText from "lucide-react-native/icons/file-text";
import { Pressable, View } from "react-native";
import { useCSSVariable } from "uniwind";

import { RelativeTime } from "@/components/RelativeTime";
import { Text } from "@/components/ui/text";
import type { DocListItem } from "@/models/Doc";

interface DocRowProps {
  doc: DocListItem;
  onPress: (doc: DocListItem) => void;
}

export const DocRow = ({ doc, onPress }: DocRowProps) => {
  const [mutedForeground] = useCSSVariable(["--color-muted-foreground"]);
  return (
    <Pressable
      role="button"
      onPress={() => onPress(doc)}
      className="min-h-14 flex-row items-center gap-3.5 px-5 py-2.5 active:bg-accent"
    >
      <FileText size={18} color={String(mutedForeground)} />
      <View className="min-w-0 flex-1 gap-0.5">
        <Text numberOfLines={2} className="text-[15px] leading-5">
          {doc.title}
        </Text>
        <Text className="font-mono text-xs text-muted-foreground">
          v{doc.version} · <RelativeTime iso={doc.updated_at} />
        </Text>
      </View>
    </Pressable>
  );
};
