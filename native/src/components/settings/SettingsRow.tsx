import ChevronRight from "lucide-react-native/icons/chevron-right";
import type { LucideIcon } from "lucide-react-native";
import { Pressable } from "react-native";
import { useCSSVariable } from "uniwind";

import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";

interface SettingsRowProps {
  label: string;
  meta?: string;
  icon?: LucideIcon;
  first?: boolean;
  onPress: () => void;
}

// A row inside a settings card: muted glyph, the label, its current value in mono, a chevron; hairlines between rows.
export const SettingsRow = ({ label, meta, icon: Icon, first = false, onPress }: SettingsRowProps) => {
  const [mutedForeground] = useCSSVariable(["--color-muted-foreground"]);
  return (
    <Pressable
      role="button"
      onPress={onPress}
      className={cn("min-h-[52px] flex-row items-center gap-3 px-4 py-3 active:bg-accent", !first && "border-t border-border")}
    >
      {Icon && <Icon size={18} color={String(mutedForeground)} />}
      <Text className="shrink-0 text-[15px]">{label}</Text>
      <Text numberOfLines={1} className="min-w-0 flex-1 text-right font-mono text-xs text-muted-foreground">
        {meta}
      </Text>
      <ChevronRight size={16} color={String(mutedForeground)} />
    </Pressable>
  );
};
