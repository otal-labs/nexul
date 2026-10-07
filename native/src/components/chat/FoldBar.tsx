import { ChevronDown, ChevronUp } from "lucide-react-native";
import { Pressable } from "react-native";
import { useCSSVariable } from "uniwind";

import { Text } from "@/components/ui/text";

interface FoldBarProps {
  label: string;
  open: boolean;
  onToggle: () => void;
}

// One hairline row that opens what a long bot post hides, then closes it again.
export const FoldBar = ({ label, open, onToggle }: FoldBarProps) => {
  const [muted] = useCSSVariable(["--color-muted-foreground"]);
  return (
    <Pressable
      role="button"
      aria-expanded={open}
      onPress={onToggle}
      className="min-h-11 flex-row items-center justify-between gap-2 rounded-md border border-border bg-card px-2.5 active:bg-accent"
    >
      <Text className="text-xs text-muted-foreground">{label}</Text>
      {open && <ChevronUp color={String(muted)} size={14} />}
      {!open && <ChevronDown color={String(muted)} size={14} />}
    </Pressable>
  );
};
