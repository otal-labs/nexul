import { ChevronRight } from "lucide-react-native";
import { Pressable } from "react-native";
import { useCSSVariable } from "uniwind";

import { Text } from "@/components/ui/text";

interface SettingsRowProps {
  label: string;
  meta?: string;
  onPress: () => void;
}

export const SettingsRow = ({ label, meta, onPress }: SettingsRowProps) => {
  const [mutedForeground] = useCSSVariable(["--color-muted-foreground"]);
  return (
    <Pressable
      role="button"
      onPress={onPress}
      className="min-h-11 flex-row items-center gap-2 border-b border-border bg-card px-4 py-3 active:bg-accent"
    >
      <Text className="shrink-0 font-medium">{label}</Text>
      <Text variant="muted" numberOfLines={1} className="min-w-0 flex-1 text-right font-mono text-xs">
        {meta}
      </Text>
      <ChevronRight size={16} color={String(mutedForeground)} />
    </Pressable>
  );
};
