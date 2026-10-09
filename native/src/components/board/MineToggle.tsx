import { Pressable } from "react-native";

import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";

interface MineToggleProps {
  on: boolean;
  onToggle: () => void;
}

// A checked control, so on it takes the ember; off it is an outline chip.
export const MineToggle = ({ on, onToggle }: MineToggleProps) => (
  <Pressable
    role="switch"
    aria-label="Only my tickets"
    aria-checked={on}
    onPress={onToggle}
    className={cn("min-h-11 justify-center self-start rounded-md border px-3.5", on ? "border-brand bg-brand" : "border-input active:bg-accent")}
  >
    <Text className={cn("text-sm font-medium", on ? "text-brand-foreground" : "text-muted-foreground")}>Only mine</Text>
  </Pressable>
);
