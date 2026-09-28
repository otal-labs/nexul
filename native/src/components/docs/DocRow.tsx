import { LockIcon } from "lucide-react-native";
import { Pressable, View } from "react-native";
import { useCSSVariable } from "uniwind";

import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";
import { formatRelativeTime } from "@/lib/time";
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
      disabled={!doc.can_open}
      onPress={() => onPress(doc)}
      className={cn(
        "min-h-11 flex-row items-center gap-2.5 border-b border-border px-4 py-3 active:bg-accent",
        !doc.can_open && "opacity-60",
      )}
    >
      <View className="min-w-0 flex-1">
        <Text numberOfLines={1} className="font-medium">
          {doc.title}
        </Text>
        <Text variant="muted" className="font-mono text-xs">
          v{doc.version} · {formatRelativeTime(doc.updated_at)}
        </Text>
      </View>
      {!doc.can_open && <LockIcon size={14} color={String(mutedForeground)} />}
    </Pressable>
  );
};
