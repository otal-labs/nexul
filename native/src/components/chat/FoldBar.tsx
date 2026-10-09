import { ChevronDown } from "lucide-react-native";
import { Pressable } from "react-native";
import { useCSSVariable } from "uniwind";

import { Text } from "@/components/ui/text";

interface FoldBarProps {
  label: string;
  open: boolean;
  onToggle: () => void;
}

// Opens what a long bot post hides, then closes it again; a ghost control whose chevron turns.
export const FoldBar = ({ label, open, onToggle }: FoldBarProps) => {
  const [muted] = useCSSVariable(["--color-muted-foreground"]);
  return (
    <Pressable
      role="button"
      aria-expanded={open}
      onPress={onToggle}
      hitSlop={8}
      className="min-h-9 flex-row items-center gap-1 self-start rounded-md px-1.5 active:bg-accent"
    >
      <Text className="text-xs font-medium text-muted-foreground">{label}</Text>
      <ChevronDown color={String(muted)} size={14} style={{ transform: [{ rotate: open ? "180deg" : "0deg" }] }} />
    </Pressable>
  );
};
